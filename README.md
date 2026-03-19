# dataAngel — Kubernetes Data Protection

> **AI-Generated Software** — This project was designed and produced by artificial intelligence (Claude / Anthropic). It is provided **as-is, with absolutely no warranty**. Use at your own risk.

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
    data-guard.io/bucket: "my-backup-bucket"
    data-guard.io/sqlite-paths: "/data/app.db"
    data-guard.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
spec:
  initContainers:
  - name: data-guard-init
    image: charchess/dataangel:0.1.0  # Pin to stable version
    command: ["./init"]
    env:
    - name: DATA_GUARD_BUCKET
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['data-guard.io/bucket']
    - name: DATA_GUARD_SQLITE_PATHS
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['data-guard.io/sqlite-paths']
    - name: DATA_GUARD_S3_ENDPOINT
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['data-guard.io/s3-endpoint']
    - name: AWS_ACCESS_KEY_ID
      valueFrom:
        secretKeyRef:
          name: data-guard-credentials
          key: access-key
    - name: AWS_SECRET_ACCESS_KEY
      valueFrom:
        secretKeyRef:
          name: data-guard-credentials
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

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Kubernetes Pod                          │
│  ┌──────────────┐    ┌─────────────────────────────────┐  │
│  │ Init Container│    │     Main Container              │  │
│  │ (restore)     │    │     (your app)                 │  │
│  └──────┬───────┘    └─────────────────────────────────┘  │
│         │                                                 │
│         ▼                                                 │
│  ┌─────────────────────────────────────────────────────┐ │
│  │              dataAngel Logic                         │ │
│  │  - Litestream restore (SQLite)                      │ │
│  │  - Rclone copy (filesystem)                         │ │
│  │  - Skip if DB exists / No replica                   │ │
│  └─────────────────────────────────────────────────────┘ │
│                           │                               │
└───────────────────────────┼───────────────────────────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │  S3 Bucket   │
                    │ (backups)     │
                    └───────────────┘
```

### Components

| Component | Purpose |
|-----------|---------|
| `init` | Restore data on pod startup if needed |
| `sidecar` | Continuous backup daemon (Litestream + Rclone) |
| `cli` | Verify backups, force-release locks |
| `metrics` | Prometheus metrics exporter |

### Monitoring

Metrics are **optional** and controlled by annotation:

```yaml
# Enable metrics (production)
annotations:
  data-guard.io/metrics-enabled: "true"

# Disable metrics (dev/CI)
annotations:
  data-guard.io/metrics-enabled: "false"
```

For Prometheus Operator auto-discovery, use the **data-guard-monitoring** component (see [kustomize/components/data-guard-monitoring](./kustomize/components/data-guard-monitoring/README.md)).

**Note:** MinIO/custom S3 endpoints are fully supported. See [issue #1](https://github.com/truxonline/dataAngel/issues/1) for implementation details.

---

## Environment Variables

### Init Container

| Variable | Required | Description |
|----------|----------|-------------|
| `DATA_GUARD_BUCKET` | Yes | S3 bucket name |
| `DATA_GUARD_SQLITE_PATHS` | No* | Comma-separated SQLite paths for Litestream restore |
| `DATA_GUARD_FS_PATHS` | No* | Comma-separated filesystem paths for Rclone restore |
| `DATA_GUARD_S3_ENDPOINT` | No | Custom S3 endpoint URL (e.g., MinIO) |
| `DATA_GUARD_FULL_LOGS` | No | Enable verbose logging (default: false) |
| `AWS_ACCESS_KEY_ID` | Yes | S3 access key (via secret) |
| `AWS_SECRET_ACCESS_KEY` | Yes | S3 secret key (via secret) |

*At least **one** of `DATA_GUARD_SQLITE_PATHS` or `DATA_GUARD_FS_PATHS` must be set.

### Sidecar Daemon

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATA_GUARD_BUCKET` | Yes | - | S3 bucket name |
| `DATA_GUARD_S3_ENDPOINT` | No | - | Custom S3 endpoint URL (e.g., MinIO) |
| `DATA_GUARD_SQLITE_PATHS` | No | - | Comma-separated SQLite paths for Litestream |
| `DATA_GUARD_FS_PATHS` | No | - | Comma-separated filesystem paths for Rclone |
| `DATA_GUARD_YAML_PATHS` | No | - | Comma-separated YAML paths to validate |
| `DATA_GUARD_RCLONE_INTERVAL` | No | `60s` | Rclone sync interval |
| `DATA_GUARD_METRICS_ENABLED` | No | `true` | Enable Prometheus metrics server |
| `DATA_GUARD_METRICS_PORT` | No | `9090` | Prometheus metrics port |
| `DATA_GUARD_SHUTDOWN_TIMEOUT` | No | `15s` | Graceful shutdown timeout |
| `AWS_ACCESS_KEY_ID` | Yes* | - | S3 access key (via secret) |
| `AWS_SECRET_ACCESS_KEY` | Yes* | - | S3 secret key (via secret) |

*Required for S3 authentication. Litestream and Rclone read these automatically.

**Usage:**

```yaml
# Create secret first
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
  containers:
  - name: myapp
    image: myapp:latest
    volumeMounts:
    - name: data
      mountPath: /data
  
  - name: dataguard-sidecar
    image: charchess/dataangel:latest
    command: ["./sidecar"]
    env:
    - name: DATA_GUARD_BUCKET
      value: "my-backup-bucket"
    - name: DATA_GUARD_SQLITE_PATHS
      value: "/data/app.db"
    - name: DATA_GUARD_FS_PATHS
      value: "/config"
    - name: DATA_GUARD_RCLONE_INTERVAL
      value: "300s"
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
    volumeMounts:
    - name: data
      mountPath: /data
    - name: config
      mountPath: /config
  
  volumes:
  - name: data
    emptyDir: {}
  - name: config
    emptyDir: {}
```

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
