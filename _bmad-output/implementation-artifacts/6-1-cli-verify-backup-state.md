# Story 6.1: CLI Verify Backup State

Status: ready-for-dev

## Story

As a Cluster Operator,
I want un CLI tool pour vérifier l'état des backups S3,
so that je peux diagnostiquer les problèmes manuellement.

## Acceptance Criteria

**Given** j'accède au CLI depuis mon poste de travail,
**When** j'exécute `data-guard-cli verify --bucket myapp`,
**Then** je vois l'état actuel des backups dans S3,
**And** je vois si des restaurations sont nécessaires.

## Tasks / Subtasks

### Task 1: CLI Command Implementation
- [ ] Add verify command to CLI
- [ ] Connect to S3
- [ ] Display backup status

### Task 2: Output Formatting
- [ ] Format readable output
- [ ] Show restore needs
- [ ] Handle errors

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `cmd/cli`
- **Dependencies**: Cobra, AWS SDK

### Source Tree Components
- `cmd/cli/verify.go`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 6.1]
- [Source: _bmad-output/planning-artifacts/prd.md#FR15: CLI verify]

### File List
- `cmd/cli/verify.go`