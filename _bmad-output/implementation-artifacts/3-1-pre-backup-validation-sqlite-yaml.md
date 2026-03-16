# Story 3.1: Pre-Backup Validation SQLite/YAML

Status: ready-for-dev

## Story

As a Cluster Operator,
I want valider l'intégrité des données avant le backup,
so that je ne sauvegarde pas des données corrompues.

## Acceptance Criteria

**Given** des données à sauvegarder,
**When** le backup est déclenché,
**Then** la validation SQLite/VAML est exécutée,
**And** le backup est bloqué si les données sont corrompues.

## Tasks / Subtasks

### Task 1: SQLite Validation
- [ ] Check database integrity (PRAGMA integrity_check)
- [ ] Verify WAL state
- [ ] Report errors

### Task 2: YAML Validation
- [ ] Parse YAML files
- [ ] Check syntax and structure
- [ ] Report errors

### Task 3: Integration with Backup
- [ ] Integrate validation in backup flow
- [ ] Block backup on validation failure
- [ ] Log validation results

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `internal/validation`
- **Dependencies**: Go YAML parser, SQLite driver

### Source Tree Components
- `internal/validation/sqlite.go`
- `internal/validation/yaml.go`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 3.1]
- [Source: _bmad-output/planning-artifacts/prd.md#FR7: Pre-backup validation]

### File List
- `internal/validation/sqlite.go`
- `internal/validation/yaml.go`
- `internal/validation/validation_test.go`