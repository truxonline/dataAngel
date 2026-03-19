# Rclone Timeout & Logging Test Procedure

## Issues Addressed

- **Issue #10**: rclone stderr not propagated to container logs
- **Issue #11**: rclone sync hangs indefinitely despite timeout flags

## Fixes Applied

### 1. Propagate stdout/stderr (Issue #10)

**Before:**
```go
func (r *realCommandRunner) Run(ctx context.Context, name string, args ...string) error {
    cmd := exec.CommandContext(ctx, name, args...)
    return cmd.Run()  // ← stderr lost
}
```

**After:**
```go
func (r *realCommandRunner) Run(ctx context.Context, name string, args ...string) error {
    cmd := exec.CommandContext(ctx, name, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr  // ← propagate stderr
    
    if err := cmd.Run(); err != nil {
        log.Printf("[%s] Command failed: %v", name, err)
        return err
    }
    return nil
}
```

**Result:** rclone errors now visible in container logs.

### 2. Add context.WithTimeout (Issue #11)

**Before:**
```go
func (r *RcloneRunner) syncOnce(ctx context.Context) error {
    // rclone has --timeout 120s but no context timeout
    return r.runner.Run(ctx, "rclone", args...)
}
```

**After:**
```go
func (r *RcloneRunner) syncOnce(ctx context.Context) error {
    // Context timeout (3min) as safety net over rclone --timeout (2min)
    syncCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
    defer cancel()
    
    return r.runner.Run(syncCtx, "rclone", args...)
}
```

**Result:** 
- rclone killed after 3min if hung
- Combined with `--timeout 120s` and `--contimeout 30s` flags for defense in depth

## Manual Test Procedure

### Test 1: Verify stderr propagation (Issue #10)

**Setup:** Point rclone to invalid endpoint

```yaml
# deployment.yaml
annotations:
  dataangel.io/s3-endpoint: "http://invalid-endpoint:9000"
```

**Expected behavior:**
- Old: `rclone sync error: exit status 1` (no details)
- New: Full rclone error visible in logs (e.g., "connection refused", "no such host")

**Commands:**
```bash
kubectl apply -k .
kubectl logs -f <pod> -c dataangel | grep rclone
```

**Success criteria:** Detailed error message from rclone visible in logs.

### Test 2: Verify timeout protection (Issue #11)

**Setup:** Simulate network hang using network policy

```yaml
# Block S3 traffic to trigger timeout
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: block-s3
  namespace: test
spec:
  podSelector:
    matchLabels:
      app: test-app
  policyTypes:
  - Egress
  egress:
  - to:
    - podSelector: {}
    ports:
    - protocol: TCP
      port: 53  # DNS only, block S3
```

**Expected behavior:**
- Old: rclone hangs indefinitely, pod stuck
- New: rclone killed after 3min, error logged, retry on next interval

**Commands:**
```bash
# Apply network policy
kubectl apply -f block-s3-policy.yaml

# Watch logs
kubectl logs -f <pod> -c dataangel | grep -E "(timeout|killed|context deadline)"

# Expected after ~3 minutes:
# [rclone] Command failed: signal: killed
# rclone sync error: context deadline exceeded
```

**Success criteria:** 
- Process terminated after 3 minutes max
- Error logged with context
- Next sync attempt happens after interval (60s)

### Test 3: Verify normal operation still works

**Setup:** Valid MinIO endpoint

```yaml
annotations:
  dataangel.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
```

**Expected behavior:**
- rclone sync succeeds
- No timeout errors
- Metrics show success

**Commands:**
```bash
kubectl apply -k .
kubectl logs -f <pod> -c dataangel | grep "rclone sync"

# Check metrics
kubectl port-forward <pod> 9090:9090
curl http://localhost:9090/metrics | grep -E "(rclone|sync)"
```

**Success criteria:**
- `dataguard_rclone_syncs_total` increments
- `dataguard_rclone_syncs_failed_total` stays at 0
- `dataguard_rclone_up` = 1

## Timeout Values Summary

| Operation | rclone flag | Context timeout | Total protection |
|-----------|-------------|-----------------|------------------|
| Sidecar sync | `--timeout 120s --contimeout 30s` | 3 minutes | ✅ Double layer |
| Init restore (rclone) | `--timeout 60s --contimeout 15s` | 2 minutes | ✅ Double layer |
| Init restore (litestream) | - | 2 minutes | ✅ Context only |

**Rationale:**
- Context timeout > rclone timeout: Let rclone clean up first
- Restore timeout < sync timeout: Init must be fast, sidecar can retry
- Defense in depth: Both rclone flags AND context timeout

## Regression Risk

**Low risk:**
- Changes are additive (timeouts + logging)
- Existing rclone flags unchanged
- Context timeout is safety net (should rarely trigger)
- stdout/stderr already captured in init container (restore.go), now consistent in sidecar

**Edge cases:**
- Large filesystem sync taking >3min: Increase timeout in code
- Slow S3 backend: rclone `--timeout` will handle, context is last resort
