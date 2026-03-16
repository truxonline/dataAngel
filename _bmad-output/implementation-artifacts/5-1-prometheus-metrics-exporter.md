# Story 5.1: Prometheus Metrics Exporter

Status: ready-for-dev

## Story

As a Cluster Operator,
I want exposer des métriques Prometheus sur les backups/restores,
so that je peux monitorer l'état du système.

## Acceptance Criteria

**Given** l'application tourne,
**When** Prometheus scrape les métriques,
**Then** les métriques dataguard_backup_duration_seconds sont exposées,
**And** les métriques dataguard_restore_operations_total sont exposées.

## Tasks / Subtasks

### Task 1: Prometheus Client Setup
- [ ] Import Prometheus Go client
- [ ] Define metrics (Counter, Gauge, Histogram)
- [ ] Register metrics

### Task 2: Metric Collection
- [ ] Update backup duration metric
- [ ] Update restore operation counter
- [ ] Expose metrics endpoint

### Task 3: Integration Tests
- [ ] Test metric exposure
- [ ] Test metric updates
- [ ] Test scrape by Prometheus

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Package**: `internal/metrics`
- **Dependencies**: Prometheus Go client

### Source Tree Components
- `internal/metrics/prometheus.go`

### References
- [Source: _bmad-output/planning-artifacts/epics.md#Story 5.1]
- [Source: _bmad-output/planning-artifacts/prd.md#FR11: Prometheus metrics]

### File List
- `internal/metrics/prometheus.go`
- `internal/metrics/prometheus_test.go`