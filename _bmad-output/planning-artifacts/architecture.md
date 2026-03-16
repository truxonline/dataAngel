---
stepsCompleted: [1, 2, 3, 4, 5, 6, 7, 8]
status: 'complete'
completedAt: '2026-03-16T11:34:28.810Z'
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - docs/specifications.md
workflowType: 'architecture'
project_name: 'dataAngel'
user_name: 'Charchess'
date: '2026-03-16'
---

# Architecture Decision Document

_This document builds collaboratively through step-by-step discovery. Sections are appended as we work through each architectural decision together._

## Project Context Analysis

### Requirements Overview

**Functional Requirements:**
16 FRs covering data restoration, backup, validation, state management, observability, configuration, and troubleshooting. Key architectural implications:
- Init container handles conditional restore logic
- Two sidecars required: Litestream (continuous SQLite backup) and Rclone (filesystem sync)
- Pre/post validation ensures data integrity
- S3 lock prevents split-brain across replicas
- Prometheus metrics and alerting for observability

**Non-Functional Requirements:**
- Performance: RTO < 2min (skip) / 5min (restore), RPO continuous/60s, Init < 30s
- Resource limits: Image < 200MB, Memory < 128MB sidecar
- Security: K8s secrets for S3 credentials, TLS support
- Integration: K8s annotations, Prometheus endpoint, S3-compatible backends

**Scale & Complexity:**
- Primary domain: Kubernetes infrastructure / Data protection
- Complexity level: Medium
- Estimated architectural components: ~8 components

**Technical Constraints & Dependencies:**
- Pure Kustomize (no Helm)
- Data paranoia: blocking behavior over data fork
- Single developer maintenance
- Simplicity over feature completeness
- K8s cluster with ArgoCD, iSCSI CSI storage, MinIO S3, Infisical secrets, Reloader

**Cross-Cutting Concerns Identified:**
- Data integrity: Validation before/after operations, WAL flush, corruption detection
- State management: S3 distributed lock with TTL + steal
- Observability: Prometheus metrics, alerting on failures/restores
- Resource constraints: Memory/CPU limits, image size optimization
- DevOps integration: K8s annotations for configuration, no external dependencies

## Starter Template Evaluation

### Primary Technology Domain

**Kubernetes Infrastructure Tool** — pas d'application web/mobile/API classique.

C'est un **système composé** avec :
- Init container (Go)
- Sidecars (Litestream, Rclone)
- CLI tool (Go)
- Metrics exporter (Prometheus)

### Selected Starter: None

**Rationale for Selection:**
Le projet nécessite des choix architecturaux spécifiques :
1. **Langage** : Go (écosystème K8s riche, populaire pour operators)
2. **Sidecars** : Litestream + Rclone (outils existants, pas de template)
3. **CLI** : Framework Go (Cobra)
4. **Metrics** : Prometheus client library

### Technical Decisions Summary

| Decision | Options | Selected |
|----------|---------|----------|
| Main Language | Go, Rust, Python | **Go** (écosystème K8s) |
| Init Container | Go, Rust | **Go** (même langage) |
| Sidecars | Litestream, Rclone | Outils existants (Go) |
| CLI Framework | Cobra, Kingpin | **Cobra** (populaire) |
| Build System | Docker, Kustomize | **Kustomize** (contraint) |
| Testing | Go test, pytest | **Go test** (native) |

## Core Architectural Decisions

### Decision Priority Analysis

**Critical Decisions (Block Implementation):**
- Langage: Go 1.22+
- Build: Docker multi-stage + distroless
- Sidecars: Init + Litestream + Rclone

**Important Decisions (Shape Architecture):**
- Observability: Prometheus
- Configuration: K8s annotations uniquement

### Data Architecture

**Language Selection:**
- **Choice:** Go 1.22+
- **Rationale:** Écosystème K8s éprouvé, concurrency native, compilation statique, image légère < 200MB
- **Affects:** Init container, sidecars, CLI, metrics exporter

### Building & Packaging

**Container Image Strategy:**
- **Choice:** Docker multi-stage avec distroless
- **Versions:** Docker 24.0+, Go 1.22
- **Rationale:** Images optimisées, dépendances minimales, sécurité maximale
- **Affects:** Tous les composants (init, sidecars)

### Sidecar Orchestration

**Deployment Pattern:**
- **Choice:** Init + 2 Sidecars (tel que spécifié dans la spec)
- **Components:**
  - Init: Restore conditionnel
  - Sidecar 1: Litestream (continuous SQLite backup)
  - Sidecar 2: Rclone (filesystem sync toutes les 60s)
- **Rationale:** Balance entre simplicité et RPO continu requis

### Monitoring & Observability

**Metrics Framework:**
- **Choice:** Prometheus
- **Rationale:** Standard K8s, écosystème riche, déjà spécifié dans la spec
- **Metrics exposés:**
  - Backup status (success/failure)
  - Restore performed (timestamp)
  - S3 sync status
  - Resource usage (memory, CPU)

### Infrastructure & Deployment

**Configuration Method:**
- **Choice:** K8s annotations uniquement (contraint spec)
- **Rationale:** Natif K8s, pas de fichier de config séparé, simple maintenance
- **Affects:** Déploiement, configuration des buckets, paths, intervals

## Implementation Patterns & Consistency Rules

### Naming Patterns

**Code Naming (Go conventions):**
- Functions: `CamelCase` — ex: `ValidateSQLiteIntegrity()`
- Variables: `camelCase` — ex: `backupStatus`
- Packages: `snake_case` (lowercase) — ex: `initcontainer`, `litestream-sidecar`
- Constants: `UPPER_SNAKE_CASE` — ex: `S3_LOCK_TTL`

**File Naming:**
- Go files: `kebab-case.go` — ex: `init-container.go`, `restore-logic.go`
- Config files: `snake_case.yaml` — ex: `kustomization.yaml`

**Kubernetes Resources:**
- Labels/Annotations: `kebab-case` — ex: `data-guard/restore-mode`
- Resource names: `kebab-case` — ex: `data-guard-init`

### Structure Patterns

**Project Organization:**

```
_data-guard/
├── cmd/
│   ├── init/                  # Init container main
│   ├── sidecar-litestream/
│   ├── sidecar-rclone/
│   └── cli/                   # CLI tool
├── internal/
│   ├── restore/               # Restore logic
│   ├── backup/                # Backup logic (Litestream/Rclone)
│   ├── validation/            # Pre/post validation
│   ├── lock/                  # S3 lock mechanism
│   └── metrics/               # Prometheus metrics
├── pkg/
│   └── k8s/                   # K8s helpers
├── kustomize/
│   └── base/                  # Kustomize manifests
└── config/
    └── data-guard.yaml        # Default config
```

### Format Patterns

**Go Code Style:**
- Imports: Grouped (standard lib, third-party, local)
- Error Handling: Always check errors, wrap with context
- Logging: Structured logging (zerolog or zap)

**Kubernetes Manifests:**
- YAML Indentation: 2 spaces
- Resource Names: `data-guard-{component}`
- Labels: App = `data-guard`, Component = `init|sidecar-litestream|sidecar-rclone`

### Communication Patterns

**Metrics Naming (Prometheus):**
- Format: `{namespace}_{metric}_{unit}`
- Examples:
  - `dataguard_backup_duration_seconds`
  - `dataguard_restore_operations_total`
  - `dataguard_lock_acquisition_failures_total`

### Process Patterns

**Error Handling:**
- Init Container: Block startup on critical errors
- Sidecars: Log and continue (don't crash pod)
- CLI: Return exit code 1 on failure

### Enforcement Guidelines

**All AI Agents MUST:**
- Use Go standard library conventions (effective Go)
- Follow Kustomize structure defined above
- Use kebab-case for Kubernetes resources
- Expose Prometheus metrics with `dataguard_` prefix

## Project Structure & Boundaries

### Complete Project Directory Structure

```
data-guard/
├── README.md
├── LICENSE
├── .gitignore
├── go.mod
├── go.sum
├── .golangci.yml
├── .github/
│   └── workflows/
│       └── ci.yml
├── cmd/
│   ├── init/
│   │   ├── main.go
│   │   ├── restore.go
│   │   ├── validate.go
│   │   └── metrics.go
│   ├── sidecar-litestream/
│   │   ├── main.go
│   │   └── litestream-wrapper.go
│   ├── sidecar-rclone/
│   │   ├── main.go
│   │   └── rclone-wrapper.go
│   └── cli/
│       ├── main.go
│       ├── verify.go
│       └── lock.go
├── internal/
│   ├── restore/
│   │   ├── restore.go
│   │   └── restore_test.go
│   ├── backup/
│   │   ├── litestream.go
│   │   ├── rclone.go
│   │   └── backup_test.go
│   ├── validation/
│   │   ├── sqlite.go
│   │   ├── yaml.go
│   │   └── validation_test.go
│   ├── lock/
│   │   ├── s3_lock.go
│   │   └── lock_test.go
│   └── metrics/
│       ├── prometheus.go
│       └── metrics_test.go
├── pkg/
│   └── k8s/
│       ├── client.go
│       ├── annotations.go
│       └── k8s_test.go
├── kustomize/
│   ├── base/
│   │   ├── deployment.yaml
│   │   ├── serviceaccount.yaml
│   │   └── kustomization.yaml
│   └── overlays/
│       ├── production/
│       └── development/
├── config/
│   └── data-guard.yaml
├── scripts/
│   ├── build.sh
│   ├── deploy.sh
│   └── test.sh
└── docs/
    └── architecture.md
```

### Architectural Boundaries

**Component Boundaries:**
- **Init Container:** Standalone, handles restore + validation, exits after execution
- **Sidecar Litestream:** Long-running, continuous SQLite backup, no shared state
- **Sidecar Rclone:** Periodic filesystem sync, no shared state with Litestream
- **CLI Tool:** External debugging tool, communicates via shared volume or S3

**Data Boundaries:**
- **S3 Storage:** Single source of truth, shared by all components
- **Local Volume:** App data only, init/sidecars handle synchronization
- **Metrics:** Prometheus endpoint on sidecar container

### Requirements to Structure Mapping

| FR Category | Implementation Location |
|-------------|------------------------|
| Data Restoration | `cmd/init/restore.go`, `internal/restore/` |
| Data Backup | `cmd/sidecar-*/`, `internal/backup/` |
| Data Validation | `internal/validation/` |
| State Management | `internal/lock/` |
| Observability | `internal/metrics/` |
| Configuration | `pkg/k8s/annotations.go` |

### Integration Points

**Internal Communication:**
- Init → Sidecars: Volume mounts, shared S3 bucket
- Sidecars → Metrics: Prometheus endpoint
- CLI → All: S3 bucket access

**External Integrations:**
- S3 (MinIO/AWS): Primary storage backend
- Prometheus: Metrics exposition
- K8s API: Annotations, secrets

**Data Flow:**
```
S3 ←→ Init (restore) ←→ Local Volume
S3 ←→ Litestream (continuous backup)
S3 ←→ Rclone (periodic sync)
Metrics ←→ Prometheus
```

## Architecture Validation Results

### Coherence Validation ✅

**Decision Compatibility:**
- Go 1.22+ compatible avec Docker multi-stage et distroless
- Prometheus compatible avec Go metrics library
- K8s annotations compatible avec Kustomize
- Toutes les versions alignées (Docker 24+, Go 1.22+)

**Pattern Consistency:**
- Naming conventions suivent Go standards
- Structure respecte le pattern Go standard
- Metrics naming follow Prometheus conventions

**Structure Alignment:**
- Project structure supporte tous les composants (init, sidecars, CLI)
- Boundaries bien définis (init exit, sidecars long-running)
- Integration points clairement spécifiés

### Requirements Coverage Validation ✅

**Functional Requirements Coverage:**

| FR Category | Arch Support | Implementation Location |
|-------------|--------------|------------------------|
| Data Restoration | ✅ | `cmd/init/restore.go`, `internal/restore/` |
| Data Backup | ✅ | `cmd/sidecar-*/`, `internal/backup/` |
| Data Validation | ✅ | `internal/validation/` |
| State Management | ✅ | `internal/lock/` |
| Observability | ✅ | `internal/metrics/` |
| Configuration | ✅ | `pkg/k8s/annotations.go` |

**Non-Functional Requirements Coverage:**
- ✅ Performance (RTO < 2min, RPO continuous/60s)
- ✅ Security (K8s secrets, TLS)
- ✅ Resource limits (Image < 200MB, Memory < 128MB)
- ✅ Integration (K8s, Prometheus, S3)

### Implementation Readiness Validation ✅

**Decision Completeness:**
- ✅ All critical decisions documented with versions
- ✅ Implementation patterns comprehensive
- ✅ Consistency rules clear and enforceable
- ✅ Examples provided for major patterns

**Structure Completeness:**
- ✅ Project structure complete and specific
- ✅ All files and directories defined
- ✅ Integration points clearly specified
- ✅ Component boundaries well-defined

**Pattern Completeness:**
- ✅ All potential conflict points addressed
- ✅ Naming conventions comprehensive
- ✅ Communication patterns fully specified
- ✅ Process patterns complete (error handling, validation flow)

### Gap Analysis Results

**Critical Gaps:** None ✅
**Important Gaps:** None ✅
**Nice-to-Have Gaps:**
- CLI usage examples (documentation)
- Kustomize overlay examples (production/development)

### Architecture Completeness Checklist

**✅ Requirements Analysis**
- [x] Project context thoroughly analyzed
- [x] Scale and complexity assessed
- [x] Technical constraints identified
- [x] Cross-cutting concerns mapped

**✅ Architectural Decisions**
- [x] Critical decisions documented with versions
- [x] Technology stack fully specified
- [x] Integration patterns defined
- [x] Performance considerations addressed

**✅ Implementation Patterns**
- [x] Naming conventions established
- [x] Structure patterns defined
- [x] Communication patterns specified
- [x] Process patterns documented

**✅ Project Structure**
- [x] Complete directory structure defined
- [x] Component boundaries established
- [x] Integration points mapped
- [x] Requirements to structure mapping complete

### Architecture Readiness Assessment

**Overall Status:** READY FOR IMPLEMENTATION

**Confidence Level:** HIGH

**Key Strengths:**
- Requirements to architecture mapping is complete
- All components have clear boundaries
- Implementation patterns are specific and enforceable
- Project structure aligns with Go/K8s best practices

**Areas for Future Enhancement:**
- CLI usage documentation
- Kustomize overlay examples
- Monitoring dashboard configuration

### Implementation Handoff

**AI Agent Guidelines:**
- Follow all architectural decisions exactly as documented
- Use implementation patterns consistently across all components
- Respect project structure and boundaries
- Refer to this document for all architectural questions

**First Implementation Priority:**
Create Go project structure and init container with restore logic
