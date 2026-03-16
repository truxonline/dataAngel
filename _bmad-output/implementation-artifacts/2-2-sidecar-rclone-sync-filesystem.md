# Story 2.2: Sidecar Rclone Sync Filesystem

Status: ready-for-dev

## Story

As a Cluster Operator,
I want le sidecar Rclone sync le filesystem vers S3 toutes les 60s,
so that les fichiers YAML et autres sont sauvegardés périodiquement.

## Acceptance Criteria

**Given** des fichiers YAML dans le répertoire de configuration,
**When** le sidecar Rclone démarre,
**Then** il sync les fichiers vers S3 toutes les 60s,
**And** les fichiers sont accessibles pour restauration.

## Tasks / Subtasks

### Task 1: Docker Image Setup
- [ ] Create Dockerfile for Rclone sidecar
- [ ] Configure Rclone sync to S3 (periodic)
- [ ] Set up logging

### Task 2: Kubernetes Sidecar Spec
- [ ] Define sidecar container spec
- [ ] Mount config volume read-only
- [ ] Configure sync interval (60s)

### Task 3: Integration Tests
- [ ] Test file sync to S3
- [ ] Verify sync interval
- [ ] Test failure recovery

## Dev Notes

### Architecture Patterns
- **Language**: Docker/Rclone
- **Package**: `cmd/sidecar-rclone`
- **Dependencies**: Rclone

### Source Tree Components
- `cmd/sidecar-rclone/Dockerfile`
- `cmd/sidecar-rclone/rclone.conf`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 2.2]
- [Source: _bmad-output/planning-artifacts/prd.md#FR5: Rclone sync]

### File List
- `cmd/sidecar-rclone/Dockerfile`
- `cmd/sidecar-rclone/rclone.conf`
- `kustomize/base/sidecar-rclone.yaml`