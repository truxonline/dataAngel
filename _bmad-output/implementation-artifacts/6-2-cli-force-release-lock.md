# Story 6.2: CLI Force Release Lock

Status: ready-for-dev

## Story

As a Cluster Operator,
I want un CLI tool pour forcer le release d'un lock bloqué,
so that je peux résoudre les problèmes de lock.

## Acceptance Criteria

**Given** un lock S3 est bloqué,
**When** j'exécute `data-guard-cli force-release-lock --bucket myapp`,
**Then** le lock est libéré immédiatement,
**And** le système peut reprendre son fonctionnement.

## Tasks / Subtasks

### Task 1: CLI Command Implementation
- [ ] Add force-release-lock command to CLI
- [ ] Connect to S3
- [ ] Delete lock object

### Task 2: Safety Checks
- [ ] Verify lock exists
- [ ] Confirm user intent
- [ ] Log operation

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `cmd/cli`
- **Dependencies**: Cobra, AWS SDK

### Source Tree Components
- `cmd/cli/lock.go`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 6.2]
- [Source: _bmad-output/planning-artifacts/prd.md#FR16: CLI force-release-lock]

### File List
- `cmd/cli/lock.go`