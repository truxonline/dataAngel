# Story 2.3: Graceful Shutdown with WAL Flush

Status: ready-for-dev

## Story

As a Cluster Operator,
I want l'application puisse s'arrêter proprement avec flush du WAL,
so that toutes les écritures sont sauvegardées avant l'arrêt.

## Acceptance Criteria

**Given** l'application en cours d'exécution,
**When** une terminaison est demandée,
**Then** le sidecar Litestream flush le WAL SQLite,
**And** toutes les écritures sont sauvegardées sur S3.

## Tasks / Subtasks

### Task 1: Signal Handling
- [ ] Intercept SIGTERM in init container
- [ ] Trigger graceful shutdown sequence

### Task 2: WAL Flush Logic
- [ ] Call Litestream flush command
- [ ] Wait for completion
- [ ] Verify S3 sync

### Task 3: Integration Tests
- [ ] Test shutdown sequence
- [ ] Verify data integrity post-shutdown

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `cmd/init` or `internal/restore`
- **Dependencies**: Litestream CLI

### Source Tree Components
- `cmd/init/shutdown.go`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 2.3]
- [Source: _bmad-output/planning-artifacts/prd.md#FR6: Graceful shutdown]

### File List
- `cmd/init/shutdown.go`