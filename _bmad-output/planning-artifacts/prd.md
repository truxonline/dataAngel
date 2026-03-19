---
stepsCompleted:
  - step-01-init
  - step-02-discovery
  - step-03-success
  - step-04-journeys
  - step-05-domain
  - step-06-innovation
  - step-07-project-type
  - step-08-scoping
  - step-09-functional
  - step-10-nonfunctional
  - step-11-polish
  - step-12-complete
inputDocuments:
  - docs/specifications.md
workflowType: 'prd'
classification:
  projectType: infrastructure_system_tool
  domain: data_protection_backup
  complexity: medium
  projectContext: greenfield
---

# Product Requirements Document - dataAngel

**Author:** Charchess
**Date:** 2026-03-16

## Executive Summary

### Vision
Data-Guard est un composant Kubernetes qui protège automatiquement les données des applications stateful avec une restauration conditionnelle, un backup continu et une alerting intégrée.

### Product Differentiator
- **Approche "just works"** — Zero interaction requise en production
- **Kubernetes-native** — Init container + sidecars, pas d'agent externe
- **S3-agnostic** — Compatible avec MinIO, AWS, et tous S3-compatible
- **Zero data loss** — RPO continu avec Litestream + Rclone

### Target Users
- **Cluster Operator** — Déploie et configure via annotations K8s
- **On-call/SRE** — Reçoit alertes, investigue via CLI
- **App Pod** — Bénéficiaire final de la protection

### Core Capabilities
- Restore automatique au démarrage si données manquantes/corrompues
- Backup continu SQLite (Litestream) et filesystem (Rclone)
- Validation intégrité avant/après backup
- Alerting sur échec restore et restauration réussie

### Scope
**MVP:** Init container + sidecars + alerting + CLI tools  
**Post-MVP:** PITR manuel via CLI  
**Known Limitations:** Pas de support PostgreSQL (mécanisme natif)

---

## Success Criteria

### User Success
- **Zéro perte de données** — Garantie que les données locales = dernières données S3
- **État cohérent** — Pod qui démarre = état du pod à l'arrêt (WAL flush, sync complete)
- **Tranquillité d'esprit** — Pas besoin de vérifier manuellement si les backup ont marché

### Business Success (Personal)
- **Maintenance minimale** — Set & forget, pas d'intervention humaine
- **Restauration fiable** — Quand besoin, restore automatique sans perte

### Technical Success
- **RTO** < 2min (skip restore) / < 5min (restore)
- **RPO** Continuous (Litestream) / 60s (Rclone filesystem)
- **Image size** < 200MB
- **Memory** < 128MB sidecar

### Measurable Outcomes
- 0 data loss incidents since deployment
- Init container < 30s when skip restore
- Pod startup blocked si S3 unavailable (by design)

## Product Scope

### MVP - Minimum Viable Product
- Init container avec restore conditionnel (si absent/corrompu/génération inférieure)
- Sidecar Litestream (continuous SQLite backup)
- Sidecar Rclone (60s filesystem sync)
- Validation SQLite/YAML avant backup
- Post-restore validation (après restauration)
- S3 lock pour éviter split-brain
- Graceful shutdown (WAL flush)
- Prometheus metrics
- CLI tools pour troubleshooting
- **Alerting** — Notifications backup failure, restore performed

### Growth Features (Post-MVP)
- (none currently defined)

### Vision (Future)
- CLI automation pour PITR (manuel)
- PostgreSQL = mécanisme natif (pas de support Data-Guard)

## Innovation Analysis

### Elicitation Insights

**First Principles**: Design simple, pas de feature creep. Init-only possible mais sidecar needed pour RPO.

**Failure Modes Identified**:
- S3 unavailable → Pod bloqué (by design, OK)
- Lock stuck → TTL + steal mechanism
- Corruption bypass → Post-restore validation added

**What If Scenarios**:
- 100GB DB → timeout configurable
- S3 data corrupt → Manual PITR only

### Improvements Applied (from Elicitation)
1. ✅ Alerting moved to MVP
2. ✅ Post-restore validation added to MVP

## User Journeys

### 1. Happy Path - Silent Operation

**User**: Cluster Operator (toi)
**Context**: Tu déploies Data-Guard sur une app (Home Assistant, Paperless)
**Flow**:
1. Tu ajoutes les annotations sur le Deployment
2. Data-guard init container vérifie l'état local vs S3
3. Si données OK → skip restore, app démarre en < 30s
4. Sidecar tourne en arrière-plan (Litestream + Rclone)
5. Tout est silent — zero interaction needed

**Emotional**: Tranquillité, "just works"

### 2. Troubleshooting - Restore Failed

**User**: Cluster Operator / On-call
**Context**: Alert "restore failed" ou pod bloqué en Init:Error
**Flow**:
1. Alert te réveille ou tu vois l'erreur
2. Tu exec dans le pod: `kubectl exec -it <pod> -c data-guard-init -- sh`
3. Tu utilises les CLI tools pour diagnostiquer
4. Tu identifies le problème (S3 down? Lock? Corruption?)
5. Tu résous (force-release-lock, check S3, etc.)
6. Tu redéploies

**Emotional**: Stress minimal car données sont safe (S3), tooling pour résoudre

### 3. Recovery Verification

**User**: Cluster Operator
**Context**: Tu suspectes un problème ou veut vérifier l'état
**Flow**:
1. Tu lances le CLI: `dataangel-cli verify --bucket myapp`
2. Tu check les metrics Prometheus
3. Tu vérifies les logs
4. Si tout OK → rien à faire
5. Si problème → manual restore avec CLI

**Emotional**: Contrôle total sur la situation

### Journey Requirements Summary

| Capability | Journey |
|------------|---------|
| K8s annotations config | Happy Path |
| Init container conditional restore | Happy Path |
| Prometheus metrics | All |
| CLI tools | Troubleshooting, Recovery |
| Graceful shutdown | Happy Path |
| S3 lock | All |
| Alerting (MVP) | Troubleshooting |
| Post-restore validation | Happy Path |

## Infrastructure Tool Specific Requirements

### Configuration
- Via K8s annotations uniquement
- Pas de fichier config séparé

### Interfaces
- Prometheus metrics (sidecar container)
- Pas de HTTP/gRPC additionnel

### CLI
- Principalement scriptable (K8s integration)
- Interactive CLI pour troubleshooting
- Output: JSON (scripts) + human-readable (logs)

## Project Scoping & Phased Development

### MVP Strategy & Philosophy

**MVP Approach:** Problem-Solving MVP — résout le problème de protection des données pour apps stateful sur K8s

**Resource Requirements:** 1 développeur (toi), skills: Go, K8s, tooling S3

### MVP Feature Set (Phase 1)

**Core User Journeys Supported:**
- Happy Path: Déploiement silencieux sans interaction
- Troubleshooting: Alerting + CLI tools
- Recovery: Vérification manuelle

**Must-Have Capabilities:**
- Init container conditional restore
- Sidecar Litestream (SQLite continuous backup)
- Sidecar Rclone (filesystem sync)
- Validation (avant + après backup)
- S3 lock
- Graceful shutdown
- Prometheus metrics
- Alerting

### Post-MVP Features

**Phase 2 (Growth):**
- (none currently defined)

**Phase 3 (Expansion):**
- CLI PITR manuel

### Risk Mitigation Strategy

**Technical Risks:** None identified — Litestream et Rclone sont des outils éprouvés

**Market Risks:** N/A — usage personnel

**Resource Risks:** N/A — projet personnel

## Functional Requirements

### Data Restoration

- FR1: **Init container** peut restaurer les données depuis S3 au démarrage du pod
- FR2: **Restore conditionnel** peut skip si données locales OK (absent/corrompu/génération inférieure)
- FR3: **Generation tracking** permet de comparer version locale vs S3

### Data Backup

- FR4: **Sidecar Litestream** peut backup SQLite en continu vers S3
- FR5: **Sidecar Rclone** peut sync filesystem vers S3 toutes les 60s
- FR6: **Graceful shutdown** peut flusher WAL avant arrest

### Data Validation

- FR7: **Pre-backup validation** peut valider intégrité SQLite/YAML avant backup
- FR8: **Post-restore validation** peut valider intégrité après restauration

### State Management

- FR9: **S3 lock** peut éviter split-brain entre replicas
- FR10: **Lock TTL + steal** peut éviter lock stuck

### Observability

- FR11: **Prometheus metrics** peut exposer backup/restore status
- FR12: **Alerting** peut notifier backup failure
- FR13: **Alerting** peut notifier restore performed

### Configuration

- FR14: **K8s annotations** peut configurer bucket, path, intervals

### Troubleshooting

- FR15: **CLI verify** peut vérifier état backup S3
- FR16: **CLI force-release-lock** peut libérer un lock stuck

## Non-Functional Requirements

### Performance

- **RTO** < 5min (skip restore) / < 5min (restore)
- **RPO** Continuous (Litestream) / 60s (Rclone)
- **Init container** < 30s quand skip restore
- **Image size** < 200MB
- **Memory** < 128MB sidecar

### Security

- **S3 credentials** via K8s secrets
- **TLS** pour S3 (S3 compatible)

### Integration

- **K8s** via annotations
- **Prometheus** metrics endpoint
- **S3** backend (tous S3-compatibles: MinIO, AWS, etc.)
