# RollingUpdate Test Procedure

## Prerequisites

- Kubernetes cluster (1.29+ for native sidecar support)
- MinIO installed in `minio` namespace
- `kubectl` CLI configured
- `watch` command available

## Test Setup

```bash
# Apply test deployment
kubectl apply -f test/rollingupdate-test.yaml

# Wait for initial deployment
kubectl wait -n dataangel-test --for=condition=ready pod -l app=test-app --timeout=120s

# Verify lock acquired
kubectl logs -n dataangel-test -l app=test-app -c dataangel | grep "Lock acquired"
```

## Test Execution

### Terminal 1: Watch Pods

```bash
watch -n 1 'kubectl get pods -n dataangel-test -o wide'
```

### Terminal 2: Watch Logs (Pod 1)

```bash
POD1=$(kubectl get pods -n dataangel-test -l app=test-app -o jsonpath='{.items[0].metadata.name}')
kubectl logs -n dataangel-test $POD1 -c dataangel -f
```

### Terminal 3: Watch Readiness

```bash
watch -n 1 'kubectl get pods -n dataangel-test -o custom-columns=NAME:.metadata.name,READY:.status.containerStatuses[0].ready,PHASE:.status.phase'
```

### Terminal 4: Trigger RollingUpdate

```bash
# Trigger update (change image tag or env var)
kubectl set image deployment/test-app -n dataangel-test app=alpine:3.20

# Or force rollout restart
kubectl rollout restart deployment/test-app -n dataangel-test

# Watch rollout status
kubectl rollout status deployment/test-app -n dataangel-test
```

## Expected Behavior

### Phase 1: Pod 2 Starts

```
t=0   Pod 1: RUNNING, lock acquired, writing to DB
t=1   RollingUpdate triggered
t=2   Pod 2: CREATED
t=3   Pod 2: Phase 1 (RESTORE) - readiness: 503 "restore in progress"
t=5   Pod 2: Restore complete
t=6   Pod 2: Phase 2 (BACKUP) - trying to acquire lock
      Pod 2: readiness: 503 "waiting for lock acquisition"
      Logs: "Failed to acquire lock (held by pod-1-xxx), retrying..."
```

### Phase 2: Pod 1 Still Running

```
t=7   Pod 1: Still RUNNING (Pod 2 not ready)
      Pod 1: Continues writing to DB
      K8s: Waits for Pod 2 readiness (does NOT terminate Pod 1 yet)
```

### Phase 3: Pod 1 Receives SIGTERM

```
t=8   K8s: Sends SIGTERM to Pod 1 (after readiness timeout or manual)
      Pod 1 logs: "Received signal: terminated"
      Pod 1: Graceful shutdown starts
      Pod 1: Lock released
      Pod 1 logs: "Lock released successfully"
```

### Phase 4: Pod 2 Acquires Lock

```
t=9   Pod 2: Lock acquisition succeeds
      Pod 2 logs: "Lock acquired, ready for traffic"
      Pod 2: SetLockAcquired(true)
      Pod 2: readiness: 200 "ok"
t=10  K8s: Marks Pod 2 READY
      K8s: Terminates Pod 1 (grace period)
```

### Phase 5: Verify No Split Brain

```
t=11  Only Pod 2 running
      Pod 2: Writing to DB
      Check S3: Only ONE active generation (no fork)
```

## Verification Steps

### 1. Check Lock State During Transition

```bash
# While Pod 2 is waiting, check S3 lock
kubectl exec -n dataangel-test $POD2 -c dataangel -- \
  sh -c 'aws s3 --endpoint-url=http://minio.minio.svc.cluster.local:9000 \
         cp s3://test-bucket/.locks/test-app - | jq .'
```

Expected output:
```json
{
  "pod_name": "test-app-6d4f8c9b7-abc12",
  "pod_uid": "...",
  "hostname": "test-app-6d4f8c9b7-abc12",
  "acquired_at": "2026-03-19T06:10:00Z",
  "ttl_seconds": 60
}
```

### 2. Verify No Concurrent Writes

```bash
# Check SQLite DB for write gaps
kubectl exec -n dataangel-test $POD2 -c app -- \
  sqlite3 /data/test.db "SELECT COUNT(*) FROM writes;"

# Check for overlapping writes (should be NONE)
kubectl exec -n dataangel-test $POD2 -c app -- \
  sqlite3 /data/test.db "
    SELECT 
      w1.hostname AS pod1, 
      w1.timestamp AS t1,
      w2.hostname AS pod2,
      w2.timestamp AS t2
    FROM writes w1, writes w2
    WHERE w1.id < w2.id
      AND w1.hostname != w2.hostname
      AND w1.timestamp > w2.timestamp
    LIMIT 5;
  "
```

Expected: **0 rows** (no concurrent writes)

### 3. Check Metrics

```bash
# Check phase transition metrics
kubectl port-forward -n dataangel-test $POD2 9090:9090 &
curl http://localhost:9090/metrics | grep dataangel_phase
```

Expected:
```
dataangel_phase{phase="restore"} 0
dataangel_phase{phase="backup"} 1
```

## Success Criteria

- ✅ Pod 2 waits for lock before becoming ready
- ✅ Pod 1 continues running while Pod 2 waits
- ✅ Pod 1 releases lock on SIGTERM
- ✅ Pod 2 acquires lock and becomes ready
- ✅ No concurrent writes detected in SQLite
- ✅ No S3 generation fork (check litestream generations)
- ✅ Zero data loss (all writes present in final DB)

## Failure Scenarios to Test

### Scenario 1: Pod 1 Crashes Without Releasing Lock

```bash
# Kill Pod 1 abruptly
kubectl delete pod -n dataangel-test $POD1 --force --grace-period=0

# Wait for TTL expiration (60s)
sleep 65

# Verify Pod 2 acquires lock after expiration
kubectl logs -n dataangel-test $POD2 -c dataangel | grep "released expired lock"
```

Expected: Pod 2 detects expiration, force-releases lock, acquires it.

### Scenario 2: Network Partition During Handoff

```bash
# Simulate network delay (requires network policy)
# Block S3 traffic temporarily to simulate slow lock operations
```

Expected: Retry loop continues until timeout (5min) or success.

## Cleanup

```bash
kubectl delete namespace dataangel-test
```

## Notes

- **Lock TTL**: Default 60s. If pod crashes, lock auto-expires.
- **Acquire timeout**: 5 minutes. Prevents infinite waiting.
- **Renewal interval**: 30s. Keeps lock alive during normal operation.
- **Readiness probe**: Checks every 2s. K8s reacts quickly to lock state changes.
