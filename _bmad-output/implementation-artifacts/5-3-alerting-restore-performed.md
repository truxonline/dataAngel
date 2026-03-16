# Story 5.3: Alerting Restore Performed

Status: ready-for-dev

## Story

As a Cluster Operator,
I want être notifié quand une restauration est effectuée,
so that je peux vérifier que le restore s'est bien passé.

## Acceptance Criteria

**Given** une restauration a été effectuée,
**When** l'alerting est configuré,
**Then** une notification est envoyée,
**And** la métrique dataguard_restore_operations_total est incrémentée.

## Tasks / Subtasks

### Task 1: Restore Detection
- [ ] Detect restore completion
- [ ] Log restore details

### Task 2: Notification Triggering
- [ ] Increment restore counter
- [ ] Send notification (Log, Webhook, etc.)
- [ ] Format notification message

### Task 3: Integration Tests
- [ ] Test restore detection
- [ ] Test notification sending
- [ ] Test metric increment

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `internal/metrics`
- **Dependencies**: Prometheus client

### Source Tree Components
- `internal/metrics/restore_notification.go`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.3]
- [Source: _bmad-output/planning-artifacts/prd.md#FR13: Alerting restore performed]

### File List
- `internal/metrics/restore_notification.go`
- `internal/metrics/restore_notification_test.go`