# dataAngel — Kubernetes Data Protection

> **AI-Generated Software** — This project was designed and produced by artificial intelligence (Claude / Anthropic). It is provided **as-is, with absolutely no warranty**. Use at your own risk.

> **⚠️ BREAKING CHANGE (v0.4.0)**: Renamed `data-guard.io` → `dataangel.io` for consistency. See [MIGRATION.md](./MIGRATION.md) for upgrade steps.

---

## What is dataAngel?

dataAngel is a Kubernetes data protection system that automatically backs up and restores your application data to/from S3. It replaces manual `rclone` setups with a clean, declarative approach using Kubernetes annotations.

### Why dataAngel?

If you're running stateful applications on Kubernetes, you need to protect your data. The options are:

| Option | Pros | Cons |
|--------|------|------|
| **Nabu Casa** | Great UI, reliable | Subscription required |
| **rclone manually** | Free | YAML config, error-prone, no restore automation |
| **dataAngel** | Free, automated, declarative | Requires S3 + annotations |

**dataAngel gives you:**
- ✅ Automatic restore on pod startup (init container)
- ✅ Continuous backup with Litestream (SQLite) or Rclone (files)
- ✅ Distributed locking to prevent concurrent backups
- ✅ Prometheus metrics for observability
- ✅ CLI tools for verification and troubleshooting

---

## Critical Setup Requirements

### SecurityContext Configuration

⚠️ **MANDATORY**: The data-guard containers (init + sidecar) **must run with the same UID/GID as your application**.

Files (SQLite DB, configs) are shared between:
- Init container (restore)
- Sidecar (continuous backup)
- Your app (read/write)

**If UIDs differ** → permission denied errors.

**Solution**: Configure Pod-level `securityContext`:

```yaml
spec:
  template:
    spec:
      securityContext:
        runAsUser: <YOUR_APP_UID>    # Match your app's UID
        runAsGroup: <YOUR_APP_GID>   # Match your app's GID
        fsGroup: <YOUR_APP_GID>      # Volume ownership
        runAsNonRoot: true           # Security best practice
```

Find your app's UID:
```bash
kubectl exec -it <existing-pod> -- id
# Output example: uid=1000(user) gid=1000(user)
```

**Common UIDs**:
- Mealie: `911:911`
- Home Assistant: `0:0` (privileged, requires `runAsNonRoot: false`)
- Vaultwarden: `1000:1000`
- Nextcloud: `33:33` (www-data)

⚠️ **Do NOT hardcode** — always check your specific app's UID.

---

## Quick Start

### 1. Choose a version

**Production**: Use pinned versions for stability
```bash
docker pull charchess/dataangel:0.1.0
```

**Development**: Use `dev` for latest main branch
```bash
docker pull charchess/dataangel:dev
```

**⚠️ Avoid `:latest` in production** — it tracks the most recent stable release and can break on automatic pulls.

See [VERSIONING.md](./VERSIONING.md) for full versioning strategy and Renovate integration.

### 2. Add annotations to your Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    dataangel.io/bucket: "my-backup-bucket"
    dataangel.io/sqlite-paths: "/data/app.db"
    dataangel.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
spec:
  initContainers:
  - name: data-guard-init
    image: charchess/dataangel:0.1.0  # Pin to stable version
    command: ["./init"]
    env:
    - name: DATA_GUARD_BUCKET
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['dataangel.io/bucket']
    - name: DATA_GUARD_SQLITE_PATHS
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['dataangel.io/sqlite-paths']
    - name: DATA_GUARD_S3_ENDPOINT
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['dataangel.io/s3-endpoint']
    - name: AWS_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: dataangel-credentials
          key: access-key
    - name: AWS_SECRET_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: dataangel-credentials
          key: secret-key
    volumeMounts:
    - name: data
      mountPath: /data
```

### 3. Use the CLI

```bash
# Verify backup state
docker run charchess/dataangel:latest ./cli verify --bucket my-backup-bucket

# Force release a stuck lock
docker run charchess/dataangel:latest ./cli force-release-lock --lock-id myapp-lock
```

---

## Architecture (v0.2.0+)

**Native Sidecar Init Container** (Kubernetes 1.29+)

```
┌─────────────────────────────────────────────────────────────┐
│                      Kubernetes Pod                          │
│                                                               │
│  ┌─────────────────────────────────────────────────────┐    │
│  │  dataangel (initContainer, restartPolicy: Always)   │    │
│  │                                                       │    │
│  │  Phase 1: RESTORE (blocks pod startup)              │    │
│  │  └─ Litestream restore (SQLite)                      │    │
│  │  └─ Rclone restore (filesystem)                      │    │
│  │                                                       │    │
│  │  Phase 2: BACKUP (runs as sidecar)                  │    │
│  │  └─ Litestream replication (continuous)             │    │
│  │  └─ Rclone sync (periodic)                          │    │
│  │  └─ Metrics server :9090 (/metrics, /ready)         │    │
│  └─────────────────────────────────────────────────────┘    │
│                           │                                   │
│  ┌─────────────────────────────────────────────────────┐    │
│  │     Main Container (your app)                       │    │
│  │     Starts only after Phase 1 completes             │    │
│  └─────────────────────────────────────────────────────┘    │
│                                                               │
└───────────────────────────┼───────────────────────────────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │  S3 Bucket   │
                    │ (backups)     │
                    └───────────────┘
```

**Key Features:**
- **1 container** instead of 2 (init + sidecar merged)
- **Phase-aware**: RESTORE blocks startup, BACKUP runs continuously
- **Auto-restart**: If backup daemon crashes, Kubernetes restarts it
- **Distributed locking**: Prevents split brain during RollingUpdates (v0.3.0+)
- **Observability**: Readiness probe (/ready), phase metrics, structured logs

### Components

| Component | Purpose |
|-----------|---------|
| `dataangel` | Unified binary: Phase 1 (restore) + Phase 2 (backup daemon) |
| `cli` | Verify backups, force-release locks |

**Phases:**
- **Phase 1 (RESTORE)**: Blocks pod startup, restores SQLite + filesystem from S3
- **Phase 2 (BACKUP)**: Runs as sidecar, continuous replication (Litestream) + periodic sync (Rclone)
  - **Lock Acquisition** (v0.3.0+): Acquires S3 distributed lock before becoming ready
  - **Heartbeat**: Renews lock every 30s to maintain ownership
  - **Graceful Shutdown**: Releases lock on SIGTERM

**Metrics Server** (port 9090):
- `/metrics` - Prometheus metrics (phase state, restore duration, sync stats)
- `/ready` - Readiness probe:
  - `503` during restore phase
  - `503` during backup phase (waiting for lock acquisition)
  - `200` after lock acquired (ready for traffic)

### Monitoring

Metrics are **optional** and controlled by annotation:

```yaml
# Enable metrics (production)
annotations:
  dataangel.io/metrics-enabled: "true"

# Disable metrics (dev/CI)
annotations:
  dataangel.io/metrics-enabled: "false"
```

For Prometheus Operator auto-discovery, use the **dataangel-monitoring** component (see [kustomize/components/dataangel-monitoring](./kustomize/components/dataangel-monitoring/README.md)).

**Note:** MinIO/custom S3 endpoints are fully supported. See [issue #1](https://github.com/truxonline/dataAngel/issues/1) for implementation details.

---

## RollingUpdate Safety (v0.3.0+)

**Problem:** SQLite apps with `strategy: RollingUpdate` can lose data during deployments due to split brain (both pods writing concurrently).

**Solution:** dataAngel uses an S3-based distributed lock to coordinate handoffs between pods:

```
RollingUpdate starts Pod 2
  ↓
Pod 2 Phase 1: Restore from S3 (gen N)
  ↓
Pod 2 Phase 2: Try to acquire S3 lock
  ↓
  Lock held by Pod 1? → Retry, NOT READY
  ↓
Kubernetes keeps Pod 1 running (Pod 2 not ready)
  ↓
Pod 1 receives SIGTERM → graceful shutdown
  ↓
Pod 1 releases lock
  ↓
Pod 2 acquires lock → READY
  ↓
Kubernetes terminates Pod 1
```

**Configuration:**

```yaml
env:
- name: DATA_GUARD_DEPLOYMENT_NAME
  value: "mealie"  # Must be unique per deployment
- name: DATA_GUARD_LOCK_TTL
  value: "60s"  # Lock expiration (prevents stuck locks)
```

**Lock Behavior:**
- **Acquisition timeout:** 5 minutes (prevents indefinite waiting)
- **Renewal interval:** 30 seconds (heartbeat keeps lock alive)
- **TTL-based expiration:** If pod crashes without releasing, lock expires after TTL
- **Force release:** Use CLI to manually release stuck locks

**Compatibility:**
- ✅ Works with any S3-compatible storage (MinIO, AWS S3, etc.)
- ✅ Zero cloud dependencies
- ✅ No Kubernetes API access required
- ✅ Self-hosted friendly

**See also:** [Issue #8](https://github.com/truxonline/dataAngel/issues/8) for detailed analysis and design discussion.

---

## Environment Variables

### dataangel Container

The unified `dataangel` container uses the same environment variables for both phases:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATA_GUARD_BUCKET` | Yes | - | S3 bucket name |
| `DATA_GUARD_SQLITE_PATHS` | No* | - | Comma-separated SQLite paths (restore + backup) |
| `DATA_GUARD_FS_PATHS` | No* | - | Comma-separated filesystem paths (restore + backup) |
| `DATA_GUARD_YAML_PATHS` | No | - | Comma-separated YAML paths to validate |
| `DATA_GUARD_S3_ENDPOINT` | No | - | Custom S3 endpoint URL (e.g., MinIO) |
| `DATA_GUARD_DEPLOYMENT_NAME` | Yes | - | Deployment name (for distributed lock) |
| `DATA_GUARD_LOCK_TTL` | No | `60s` | Lock expiration timeout |
| `DATA_GUARD_LOCK_ENABLED` | No | `true` | Disable distributed lock (single-replica) |
| `DATA_GUARD_LOCK_ACQUIRE_TIMEOUT` | No | `5m` | Max wait to acquire the lock on startup |
| `DATA_GUARD_RESTORE_TIMEOUT` | No | `30m` | Max duration for a single restore operation |
| `DATA_GUARD_RESTORE_OVERWRITE` | No | `false` | If `true`, overwrite newer local files during restore (disables `--update`) |
| `DATA_GUARD_RCLONE_INTERVAL` | No | `60s` | Rclone sync interval |
| `DATA_GUARD_RCLONE_DELAY` | No | `30s` | Initial delay before first rclone sync |
| `DATA_GUARD_RCLONE_TRANSFERS` | No | `1` | Rclone parallel transfer count |
| `DATA_GUARD_RCLONE_CHECKERS` | No | `2` | Rclone parallel checker count |
| `DATA_GUARD_RCLONE_BWLIMIT` | No | - | Rclone bandwidth limit (e.g., `10M`) |
| `DATA_GUARD_SYNC_TIMEOUT` | No | `3m` | Timeout per rclone sync operation |
| `DATA_GUARD_EXCLUDE_PATTERNS` | No | `*.db*,.*.db-litestream/**,*.dataangel-clean` | Comma-separated rclone exclude patterns |
| `DATA_GUARD_METRICS_ENABLED` | No | `true` | Enable Prometheus metrics server |
| `DATA_GUARD_METRICS_PORT` | No | `9090` | Prometheus metrics port |
| `DATA_GUARD_SHUTDOWN_TIMEOUT` | No | `15s` | Graceful shutdown timeout |
| `DATA_GUARD_FULL_LOGS` | No | `false` | Enable verbose logging |
| `AWS_ACCESS_KEY_ID` | Yes | - | S3 access key (via secret) |
| `AWS_SECRET_ACCESS_KEY` | Yes | - | S3 secret key (via secret) |

*At least **one** of `DATA_GUARD_SQLITE_PATHS` or `DATA_GUARD_FS_PATHS` must be set.

**Usage Example:**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dataangel-s3-creds
type: Opaque
stringData:
  access-key: "YOUR_ACCESS_KEY"
  secret-key: "YOUR_SECRET_KEY"
---
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  initContainers:
  - name: dataangel
    restartPolicy: Always
    image: charchess/dataangel:0.2.0
    command: ["./dataangel"]
    env:
    - name: DATA_GUARD_BUCKET
      value: "my-backup-bucket"
    - name: DATA_GUARD_SQLITE_PATHS
      value: "/data/app.db"
    - name: DATA_GUARD_FS_PATHS
      value: "/config"
    - name: DATA_GUARD_RCLONE_INTERVAL
      value: "300s"
    - name: DATA_GUARD_METRICS_ENABLED
      value: "true"
    - name: AWS_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: dataangel-s3-creds
          key: access-key
    - name: AWS_SECRET_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: dataangel-s3-creds
          key: secret-key
    ports:
    - containerPort: 9090
      name: metrics
    readinessProbe:
      httpGet:
        path: /ready
        port: 9090
      initialDelaySeconds: 0
      periodSeconds: 2
    volumeMounts:
    - name: data
      mountPath: /data
    - name: config
      mountPath: /config
  
  containers:
  - name: myapp
    image: myapp:latest
    volumeMounts:
    - name: data
      mountPath: /data
  
  volumes:
  - name: data
    emptyDir: {}
  - name: config
    emptyDir: {}
```

**Note:** `restartPolicy: Always` on the init container makes it behave as a sidecar after Phase 1 completes. Requires Kubernetes 1.29+.

### CLI

| Command | Description |
|---------|-------------|
| `verify --bucket <name>` | List backups in S3 bucket |
| `force-release-lock --lock-id <id>` | Release a stuck distributed lock |

---

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success (data already up-to-date or restored) |
| 1 | Restore needed but failed |
| 2 | Configuration error (missing env vars) |

---

## Roadmap

- [ ] Webhook for automatic pod mutation
- [ ] Kustomize overlays for common patterns
- [ ] Helm chart
- [ ] Backup scheduling

---

## License

MIT — see LICENSE for details.

---

## Disclaimer

> This software is AI-generated and provided without warranty of any kind.
> - It has not been audited for security or reliability.
> - Always test with non-production data first.
> - Maintain backups of your S3 data.
>
> By using dataAngel, you acknowledge these risks.
