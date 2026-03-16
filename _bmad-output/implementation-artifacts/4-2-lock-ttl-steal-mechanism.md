# Story 4.2: Lock TTL + Steal Mechanism

Status: ready-for-dev

## Story

As a Cluster Operator,
I want un TTL sur les locks S3 avec mécanisme de steal,
so that un lock bloqué peut être libéré automatiquement.

## Acceptance Criteria

**Given** un lock S3 a été acquis,
**When** le TTL expire sans release,
**Then** un autre replica peut voler le lock,
**And** le lock est libéré pour le nouveau propriétaire.

## Tasks / Subtasks

### Task 1: TTL Implementation
- [ ] Add timestamp to lock object
- [ ] Implement TTL check logic
- [ ] Auto-expire stale locks

### Task 2: Steal Mechanism
- [ ] Check for expired locks
- [ ] Acquire lock if expired
- [ ] Clean up old lock data

### Task 3: Integration Tests
- [ ] Test TTL expiration
- [ ] Test lock stealing
- [ ] Test concurrent steal prevention

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `internal/lock`
- **Dependencies**: AWS SDK

### Source Tree Components
- `internal/lock/ttl.go`
- `internal/lock/steal.go`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 4.2]
- [Source: _bmad-output/planning-artifacts/prd.md#FR10: Lock TTL + steal]

### File List
- `internal/lock/ttl.go`
- `internal/lock/steal.go`
- `internal/lock/ttl_steal_test.go`