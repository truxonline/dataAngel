# Story 1.2: Init Container Detect Healthy Data

Status: ready-for-dev

## Story

As a Cluster Operator,
I want l'init container puisse vérifier l'état local vs S3,
so that il décide si un restore est nécessaire.

## Acceptance Criteria

**Given** des données locales présentes sur le volume,
**When** l'init container démarre et consulte S3,
**Then** il compare la version locale avec la version S3,
**And** il détermine si les données sont valides, corrompues, ou manquantes.

## Tasks / Subtasks

### Task 1: Read Local State
- [ ] Identify local data directory
- [ ] Read generation/metadata files
- [ ] Compute local checksum/validation

### Task 2: Fetch S3 State
- [ ] List objects in S3 bucket path
- [ ] Read S3 metadata/generation file
- [ ] Compute S3 checksum/validation

### Task 3: Compare States
- [ ] Compare local vs S3 generation
- [ ] Identify if local is ahead, behind, or missing
- [ ] Determine restore necessity (missing/corrupt)

### Task 4: Output Decision
- [ ] Log comparison results
- [ ] Set exit code/signal for next step

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `internal/restore`
- **Dependencies**: `pkg/k8s`, AWS SDK (S3)

### Source Tree Components
- `cmd/init/` - Init container entrypoint
- `internal/restore/state_check.go` - State comparison logic
- `pkg/s3/` - S3 interaction helpers

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 1.2]
- [Source: _bmad-output/planning-artifacts/prd.md#FR2: Restore conditionnel]

### File List
- `cmd/init/main.go` - Entry point (calls state check)
- `internal/restore/state_check.go` - State comparison logic
- `internal/restore/state_check_test.go` - Tests