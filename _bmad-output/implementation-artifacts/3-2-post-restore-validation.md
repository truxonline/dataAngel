# Story 3.2: Post-Restore Validation

Status: ready-for-dev

## Story

As a Cluster Operator,
I want valider l'intégrité des données après la restauration,
so that je confirme que les données restaurées sont valides.

## Acceptance Criteria

**Given** une restauration vient d'être effectuée,
**When** l'init container vérifie les données,
**Then** une validation d'intégrité est exécutée,
**And** une alerte est déclenchée si les données sont invalides.

## Tasks / Subtasks

### Task 1: Post-Restore Validation Logic
- [ ] Run SQLite integrity check
- [ ] Verify YAML syntax
- [ ] Compare checksums if available

### Task 2: Alerting on Failure
- [ ] Trigger alert if validation fails
- [ ] Log error details
- [ ] Prevent pod startup if critical

### Task 3: Integration Tests
- [ ] Test validation after restore
- [ ] Test alert triggering
- [ ] Test recovery scenarios

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `internal/validation`
- **Dependencies**: Same as pre-backup validation

### Source Tree Components
- `internal/validation/post_restore.go`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 3.2]
- [Source: _bmad-output/planning-artifacts/prd.md#FR8: Post-restore validation]

### File List
- `internal/validation/post_restore.go`
- `internal/validation/post_restore_test.go`