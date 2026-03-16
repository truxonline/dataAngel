# Story 5.2: Alerting Backup Failure

Status: ready-for-dev

## Story

As a Cluster Operator,
I want recevoir une alerte si le backup échoue,
so that je peux intervenir rapidement.

## Acceptance Criteria

**Given** un backup échoue (S3 indisponible, etc.),
**When** l'alerting est configuré,
**Then** une alerte est envoyée,
**And** la métrique dataguard_backup_failures_total est incrémentée.

## Tasks / Subtasks

### Task 1: Failure Detection
- [ ] Detect S3 connection failures
- [ ] Detect backup errors
- [ ] Log failure details

### Task 2: Alert Triggering
- [ ] Increment failure counter
- [ ] Send alert (Log, Webhook, etc.)
- [ ] Format alert message

### Task 3: Integration Tests
- [ ] Test failure detection
- [ ] Test alert sending
- [ ] Test metric increment

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `internal/metrics`
- **Dependencies**: Prometheus client

### Source Tree Components
- `internal/metrics/alerting.go`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.2]
- [Source: _bmad-output/planning-artifacts/prd.md#FR12: Alerting backup failure]

### File List
- `internal/metrics/alerting.go`
- `internal/metrics/alerting_test.go`