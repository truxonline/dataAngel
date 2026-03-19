#!/bin/bash
# Commands to close issues #6 and #7
# Run manually: bash .github/CLOSE_ISSUES.sh

echo "Closing issue #6 (implemented)..."
gh issue close 6 -c "Implemented in v0.2.0 (commit 61f2cfa).

Native sidecar init container with restartPolicy: Always.
Phase-aware execution with metrics and readiness probe.
Requires Kubernetes 1.29+.

Changes:
- Unified dataangel binary (1 container replaces init+sidecar)
- Phase 1 (RESTORE): Blocks pod startup, exits on failure
- Phase 2 (BACKUP): Runs as sidecar daemon
- Observability: logs, metrics (dataangel_phase, dataangel_restore_duration_seconds), readiness probe (/ready)
- Kustomize component simplified (1 patch instead of 2)

Migration: K8s >= 1.29 required. Update component image to 0.2.0."

echo "Closing issue #7 (rejected)..."
gh issue close 7 --reason "not planned" -c "Decision: **REJECTED** for architectural reasons.

shareProcessNamespace adds complexity with minimal benefit for dataAngel:
- **Breaks agnostic design**: Requires user to add shareProcessNamespace: true (not transparent)
- **Weak use case**: Monitoring app crashes doesn't help backup reliability
- **Orthogonal to mission**: dataAngel focuses on data protection, not process supervision

Alternative: Use Kubernetes native pod health monitoring (livenessProbe, readinessProbe) for app health.

dataAngel v0.2.0 already includes readiness probe for phase visibility."

echo "Done. Issues closed."
