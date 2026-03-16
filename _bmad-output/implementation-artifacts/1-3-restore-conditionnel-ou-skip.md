# Story 1.3: Restore Conditionnel ou Skip

Status: ready-for-dev

## Story

As a Cluster Operator,
I want l'init container puisse restaurer ou skip le restore,
so that le pod démarre rapidement si les données sont valides.

## Acceptance Criteria

**Given** l'init container a déterminé l'état des données,
**When** les données sont valides et plus récentes que S3,
**Then** l'init container skip le restore et continue,
**And** le pod démarre en < 30s.

## Tasks / Subtasks

### Task 1: Implement Restore Logic
- [ ] Fetch data from S3 if needed
- [ ] Verify downloaded data integrity
- [ ] Restore to local volume

### Task 2: Implement Skip Logic
- [ ] Check if local data is valid
- [ ] Verify local data is not older than S3
- [ ] Bypass restore process

### Task 3: Performance Optimization
- [ ] Optimize S3 fetch (streaming, parallel)
- [ ] Minimize pod startup time
- [ ] Ensure < 30s startup when skipping

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `internal/restore`
- **Dependencies**: `pkg/s3`, `internal/validation`

### Source Tree Components
- `cmd/init/restore.go` - Restore logic
- `internal/restore/restore.go` - Core restore implementation
- `internal/restore/skip.go` - Skip logic

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.3]
- [Source: _bmad-output/planning-artifacts/prd.md#FR2: Restore conditionnel]

### File List
- `cmd/init/restore.go` - Restore entry point
- `internal/restore/restore.go` - Core implementation
- `internal/restore/skip.go` - Skip logic
- `internal/restore/restore_test.go` - Tests