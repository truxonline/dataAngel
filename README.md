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

## Quick Start

### 1. Deploy the image

```bash
# Pull the latest image
docker pull charchess/dataangel:latest
```

### 2. Add annotations to your Pod

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
  annotations:
    dataguard/enable: "true"
    dataguard/bucket: "my-backup-bucket"
    dataguard/path: "data/app.db"
    dataguard/checksum: "sha256:abc123..."
spec:
  initContainers:
  - name: dataguard-restore
    image: charchess/dataangel:latest
    command: ["./init"]
    env:
    - name: DATA_GUARD_BUCKET
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['dataguard/bucket']
    - name: DATA_GUARD_PATH
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['dataguard/path']
    - name: DATA_GUARD_LOCAL_PATH
      value: "/data"
    - name: DATA_GUARD_CHECKSUM
      valueFrom:
        fieldRef:
          fieldPath: metadata.annotations['dataguard/checksum']
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
│  │  - Check local vs S3 state                          │ │
│  │  - Conditional restore                              │ │
│  │  - Checksum validation                              │ │
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

---

## Environment Variables

### Init Container

| Variable | Required | Description |
|----------|----------|-------------|
| `DATA_GUARD_BUCKET` | Yes | S3 bucket name |
| `DATA_GUARD_PATH` | Yes | Path in bucket (e.g., `backups/db.sqlite`) |
| `DATA_GUARD_LOCAL_PATH` | Yes | Local path to restore to |
| `DATA_GUARD_CHECKSUM` | Yes | Expected SHA256 checksum |
| `DATA_GUARD_AWS_REGION` | No | AWS region (default: `us-east-1`) |

### Sidecar Daemon

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATA_GUARD_BUCKET` | Yes | - | S3 bucket name |
| `DATA_GUARD_S3_ENDPOINT` | No | - | Custom S3 endpoint URL |
| `DATA_GUARD_SQLITE_PATHS` | No | - | Comma-separated SQLite paths for Litestream |
| `DATA_GUARD_FS_PATHS` | No | - | Comma-separated filesystem paths for Rclone |
| `DATA_GUARD_YAML_PATHS` | No | - | Comma-separated YAML paths to validate |
| `DATA_GUARD_RCLONE_INTERVAL` | No | `60s` | Rclone sync interval |
| `DATA_GUARD_METRICS_PORT` | No | `9090` | Prometheus metrics port |
| `DATA_GUARD_SHUTDOWN_TIMEOUT` | No | `15s` | Graceful shutdown timeout |

**Usage:**

```yaml
containers:
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
  ports:
  - containerPort: 9090
    name: metrics
  volumeMounts:
  - name: data
    mountPath: /data
  - name: config
    mountPath: /config
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
