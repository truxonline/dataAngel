# Sidecar Daemon

The sidecar daemon runs alongside your application in a Kubernetes pod to continuously back up data to S3.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      Kubernetes Pod                          │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Sidecar Container                        │  │
│  │  ┌────────────────────────────────────────────────┐  │  │
│  │  │  Daemon (errgroup)                             │  │  │
│  │  │  ├─ Litestream Replicator (SQLite)            │  │  │
│  │  │  ├─ Rclone Sync Loop (Files/YAML)             │  │  │
│  │  │  └─ Metrics Server (Prometheus)               │  │  │
│  │  └────────────────────────────────────────────────┘  │  │
│  └──────────────────────────────────────────────────────┘  │
│                           │                                 │
└───────────────────────────┼─────────────────────────────────┘
                            │
                            ▼
                    ┌───────────────┐
                    │  S3 Bucket   │
                    │ (backups)     │
                    └───────────────┘
```

## Components

### Litestream Replicator
- Continuously replicates SQLite databases to S3
- One replicator per database path
- Graceful shutdown with 15s WAL flush timeout
- Environment: `LITESTREAM_S3_*` variables

### Rclone Sync Loop
- Syncs file and YAML paths to S3 on interval
- Validates YAML patterns before sync
- Respects context cancellation
- Configurable sync interval (default: 60s)

### Metrics Server
- Prometheus metrics on port 9090
- Tracks backup success/failure rates
- Monitors daemon health

## Configuration

All configuration via environment variables:

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `DATA_GUARD_BUCKET` | Yes | - | S3 bucket name |
| `DATA_GUARD_S3_ENDPOINT` | No | - | S3 endpoint (e.g., http://minio:9000) |
| `DATA_GUARD_SQLITE_PATHS` | No | - | CSV of SQLite paths to replicate |
| `DATA_GUARD_FS_PATHS` | No | - | CSV of filesystem paths to sync |
| `DATA_GUARD_YAML_PATHS` | No | - | CSV of YAML glob patterns to sync |
| `DATA_GUARD_RCLONE_INTERVAL` | No | 60 | Sync interval in seconds |
| `DATA_GUARD_METRICS_PORT` | No | 9090 | Prometheus metrics port |
| `DATA_GUARD_SHUTDOWN_TIMEOUT` | No | 15 | Graceful shutdown timeout in seconds |

## Usage

### Docker

```bash
docker run -e DATA_GUARD_BUCKET=my-bucket \
           -e DATA_GUARD_S3_ENDPOINT=http://minio:9000 \
           -e DATA_GUARD_SQLITE_PATHS=/data/app.db \
           -e DATA_GUARD_FS_PATHS=/config \
           -v /data:/data \
           -v /config:/config \
           charchess/dataangel:sidecar
```

### Kubernetes

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: myapp
spec:
  containers:
  - name: app
    image: myapp:latest
    volumeMounts:
    - name: data
      mountPath: /data
    - name: config
      mountPath: /config

  - name: sidecar
    image: charchess/dataangel:sidecar
    env:
    - name: DATA_GUARD_BUCKET
      value: "my-backup-bucket"
    - name: DATA_GUARD_S3_ENDPOINT
      value: "http://minio:9000"
    - name: DATA_GUARD_SQLITE_PATHS
      value: "/data/app.db"
    - name: DATA_GUARD_FS_PATHS
      value: "/config"
    - name: DATA_GUARD_YAML_PATHS
      value: "/config/*.yaml"
    - name: DATA_GUARD_RCLONE_INTERVAL
      value: "60"
    - name: DATA_GUARD_METRICS_PORT
      value: "9090"
    volumeMounts:
    - name: data
      mountPath: /data
    - name: config
      mountPath: /config
    ports:
    - name: metrics
      containerPort: 9090
    livenessProbe:
      httpGet:
        path: /metrics
        port: 9090
      initialDelaySeconds: 10
      periodSeconds: 30

  volumes:
  - name: data
    emptyDir: {}
  - name: config
    configMap:
      name: app-config
```

## Graceful Shutdown

The daemon handles SIGINT and SIGTERM signals:

1. Signal received → context cancelled
2. All goroutines notified via context
3. Litestream: 15s timeout for WAL flush
4. Rclone: immediate stop on next tick
5. Metrics: graceful HTTP shutdown
6. Exit with status 0

## Testing

```bash
cd internal/sidecar
go test -v ./...
```

Test coverage includes:
- Config loading from environment
- Litestream command building and execution
- Rclone sync loop with ticker
- Daemon lifecycle with errgroup
- Signal handling and graceful shutdown
- YAML path validation
- CSV parsing with whitespace trimming

## Metrics

Available at `http://localhost:9090/metrics`:

```
dataguard_litestream_replications_total
dataguard_litestream_replication_errors_total
dataguard_rclone_syncs_total
dataguard_rclone_sync_errors_total
dataguard_daemon_uptime_seconds
```

## Troubleshooting

### "litestream: executable file not found"
- Ensure litestream is installed in the container
- Check Dockerfile includes `apk add litestream`

### "rclone: executable file not found"
- Ensure rclone is installed in the container
- Check Dockerfile includes `apk add rclone`

### Sync not happening
- Check `DATA_GUARD_RCLONE_INTERVAL` is set correctly
- Verify S3 credentials are available to rclone
- Check logs for rclone errors

### High memory usage
- Reduce `DATA_GUARD_RCLONE_INTERVAL` to sync less frequently
- Check for large files in sync paths
- Monitor with `curl http://localhost:9090/metrics`

## Development

### Building locally

```bash
cd cmd/sidecar
go build -o sidecar .
```

### Running with test config

```bash
DATA_GUARD_BUCKET=test-bucket \
DATA_GUARD_S3_ENDPOINT=http://localhost:9000 \
DATA_GUARD_SQLITE_PATHS=/tmp/test.db \
./sidecar
```

### Running tests with coverage

```bash
cd internal/sidecar
go test -v -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Performance

- Litestream: Continuous replication (near real-time)
- Rclone: Interval-based sync (default 60s)
- Memory: ~50MB base + sync buffer
- CPU: Minimal when idle, spikes during sync

## Security

- No hardcoded credentials
- All config via environment variables
- S3 credentials via standard AWS SDK chain
- Rclone config via environment or files
- Graceful signal handling (no SIGKILL)

## License

MIT — see LICENSE for details.
