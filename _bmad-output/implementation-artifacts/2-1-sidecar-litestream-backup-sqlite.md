# Story 2.1: Sidecar Litestream Backup SQLite

Status: done

## Story

As a Cluster Operator,
I want le sidecar Litestream backup SQLite en continu vers S3,
so that mes données sont sauvegardées en temps réel.

## Acceptance Criteria

**Given** une application avec base SQLite,
**When** le sidecar Litestream démarre,
**Then** il stream les modifications SQLite vers S3 en continu,
**And** les données sont accessibles pour restauration.

## Tasks / Subtasks

### Task 1: Docker Image Setup
- [x] Create Dockerfile for Litestream sidecar
- [x] Configure Litestream replication to S3
- [x] Set up logging and health checks

### Task 2: Kubernetes Sidecar Spec
- [x] Define sidecar container spec
- [x] Mount SQLite volume read-only
- [x] Configure S3 credentials via env/secret

### Task 3: Integration Tests
- [x] Test SQLite streaming to S3
- [x] Verify data integrity post-backup
- [x] Test failure recovery

## Dev Notes

### Architecture Patterns
- **Language**: Docker/Litestream (Go based)
- **Package**: `cmd/sidecar-litestream`
- **Dependencies**: Litestream, AWS SDK

### Source Tree Components
- `cmd/sidecar-litestream/Dockerfile`
- `cmd/sidecar-litestream/litestream.yml` (config)

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 2.1]
- [Source: _bmad-output/planning-artifacts/prd.md#FR4: Litestream backup]

### File List
- `cmd/sidecar-litestream/Dockerfile`
- `cmd/sidecar-litestream/dockerfile.go` (updated: error handling)
- `cmd/sidecar-litestream/dockerfile_test.go` (updated: error handling)
- `cmd/sidecar-litestream/integration_test.go` (cleaned: removed performance test)
- `cmd/sidecar-litestream/k8s_spec.go` (updated: YAML indentation, error handling)
- `cmd/sidecar-litestream/k8s_spec_test.go` (updated: error handling)
- `cmd/sidecar-litestream/litestream.yml`
- `cmd/sidecar-litestream/s3_streaming.go`
- `kustomize/base/sidecar-litestream.yaml`