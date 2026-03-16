# Story 1.1: Configurer Data-Guard via Annotations K8s

Status: ready-for-dev

## Story

As a Cluster Operator,
I want add Data-Guard annotations on a Kubernetes Deployment,
so that Data-Guard can be deployed automatically on my application.

## Acceptance Criteria

**Given** a Kubernetes Deployment for Home Assistant with data volume mounted,
**When** I add the following annotations to the Deployment:
- `data-guard/bucket: my-bucket`
- `data-guard/path: /home-assistant`
- `data-guard/backup-interval: 60`
**Then** the Data-Guard init container and sidecars are automatically injected into the pod
**And** the init container receives the correct S3 bucket and path configuration
**And** the sidecars receive the correct backup interval configuration
**And** the pod starts successfully with Data-Guard components running

## Tasks / Subtasks

### Task 1: Create K8s Annotation Parsing Logic
- [ ] Define annotation constants in Go
- [ ] Create annotation parsing function
- [ ] Handle missing/invalid annotations gracefully

### Task 2: Create Init Container Spec Generator
- [ ] Generate init container spec from annotations
- [ ] Mount required volumes
- [ ] Pass configuration via environment variables

### Task 3: Create Sidecar Spec Generators
- [ ] Generate Litestream sidecar spec from annotations
- [ ] Generate Rclone sidecar spec from annotations
- [ ] Mount shared volumes

### Task 4: Integration Tests
- [ ] Test annotation parsing
- [ ] Test container injection
- [ ] Test configuration propagation

## Dev Notes

### Architecture Patterns
- **Language**: Go 1.22+
- **Framework**: Standard library + k8s client-go
- **Image**: Distroless base
- **Memory**: < 128MB per sidecar

### Source Tree Components
- `cmd/init/` - Init container entrypoint
- `internal/validation/` - Configuration validation
- `pkg/k8s/` - K8s helper functions

### K8s Resource Structure
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: home-assistant
  annotations:
    data-guard/bucket: my-bucket
    data-guard/path: /home-assistant
    data-guard/backup-interval: "60"
spec:
  template:
    spec:
      initContainers:
        - name: data-guard-init
      containers:
        - name: home-assistant
        - name: data-guard-litestream
        - name: data-guard-rclone
```

### Annotations to Parse
| Annotation | Type | Required | Description |
|------------|------|----------|-------------|
| `data-guard/bucket` | string | Yes | S3 bucket name |
| `data-guard/path` | string | Yes | Backup path in bucket |
| `data-guard/backup-interval` | int | Yes | Rclone sync interval (seconds) |

### Configuration Passing
- Init container: Environment variables `DATA_GUARD_BUCKET`, `DATA_GUARD_PATH`
- Sidecars: Environment variables passed via container spec

## References

- [Source: _bmad-output/planning-artifacts/epics.md#Epic 1: Initial Setup & Data Discovery]
- [Source: _bmad-output/planning-artifacts/architecture.md#Project Structure]
- [Source: _bmad-output/planning-artifacts/prd.md#FR14: K8s annotations]

## Dev Agent Record

### Agent Model Used
mimo-v2-flash-free

### Debug Log References
- Annotation parsing: TBD
- Container injection: TBD

### Completion Notes List
- [ ] Create annotation parsing logic
- [ ] Create init container spec
- [ ] Create sidecar specs
- [ ] Write integration tests
- [ ] Verify pod starts successfully

### File List
- `cmd/init/main.go` - Init container entrypoint
- `internal/validation/annotations.go` - Annotation parsing
- `pkg/k8s/spec.go` - Container spec generation
- `pkg/k8s/spec_test.go` - Spec generation tests