# Story 1.4: CLI Verify Backup State

Status: ready-for-dev

## Story

As a Cluster Operator,
I want un CLI tool pour vérifier l'état du backup S3,
so that je peux diagnostiquer les problèmes manuellement.

## Acceptance Criteria

**Given** j'accède au CLI depuis mon poste de travail,
**When** j'exécute `dataangel-cli verify --bucket myapp`,
**Then** je vois l'état actuel des backups dans S3,
**And** je vois si des restaurations sont nécessaires.

## Tasks / Subtasks

### Task 1: CLI Framework Setup
- [ ] Setup Cobra CLI framework
- [ ] Define command structure (verify, force-release-lock)
- [ ] Add flags (bucket, path, verbose)

### Task 2: Verify Command Logic
- [ ] Connect to S3
- [ ] List backup objects
- [ ] Display status to user

### Task 3: User Output
- [ ] Format readable output
- [ ] Show restore status
- [ ] Handle errors gracefully

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `cmd/cli`
- **Dependencies**: Cobra, AWS SDK

### Source Tree Components
- `cmd/cli/main.go` - CLI entrypoint
- `cmd/cli/verify.go` - Verify command

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.4]
- [Source: _bmad-output/planning-artifacts/prd.md#FR15: CLI verify]

### File List
- `cmd/cli/main.go`
- `cmd/cli/verify.go`