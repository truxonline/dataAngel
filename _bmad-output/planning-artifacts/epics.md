---
stepsCompleted: [step-01-validate-prerequisites, step-02-design-epics, step-03-create-stories, step-04-final-validation]
inputDocuments:
  - _bmad-output/planning-artifacts/prd.md
  - _bmad-output/planning-artifacts/architecture.md
---

# dataAngel - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for dataAngel, decomposing the requirements from the PRD, Architecture requirements into implementable stories.

## Requirements Inventory

### Functional Requirements

FR1: **Init container** peut restaurer les données depuis S3 au démarrage du pod
FR2: **Restore conditionnel** peut skip si données locales OK (absent/corrompu/génération inférieure)
FR3: **Generation tracking** permet de comparer version locale vs S3
FR4: **Sidecar Litestream** peut backup SQLite en continu vers S3
FR5: **Sidecar Rclone** peut sync filesystem vers S3 toutes les 60s
FR6: **Graceful shutdown** peut flusher WAL avant arrest
FR7: **Pre-backup validation** peut valider intégrité SQLite/YAML avant backup
FR8: **Post-restore validation** peut valider intégrité après restauration
FR9: **S3 lock** peut éviter split-brain entre replicas
FR10: **Lock TTL + steal** peut éviter lock stuck
FR11: **Prometheus metrics** peut exposer backup/restore status
FR12: **Alerting** peut notifier backup failure
FR13: **Alerting** peut notifier restore performed
FR14: **K8s annotations** peut configurer bucket, path, intervals
FR15: **CLI verify** peut vérifier état backup S3
FR16: **CLI force-release-lock** peut libérer un lock stuck

### Non-Functional Requirements

NFR1: RTO < 2min (skip restore) / 5min (restore)
NFR2: RPO Continuous (Litestream) / 60s (Rclone)
NFR3: Init container < 30s when skip restore
NFR4: Image size < 200MB
NFR5: Memory < 128MB sidecar
NFR6: S3 credentials via K8s secrets
NFR7: TLS for S3 (S3 compatible)
NFR8: K8s via annotations
NFR9: Prometheus metrics endpoint
NFR10: S3 backend (all S3-compatible)

### Additional Requirements (from Architecture)

- Pure Kustomize (no Helm) implementation
- Data paranoia: blocking behavior preferred over data fork
- Single developer maintenance
- Simplicity over feature completeness
- Go 1.22+ language
- Docker multi-stage + distroless build
- Init + 2 Sidecars deployment pattern
- Prometheus metrics framework
- K8s annotations configuration only

### FR Coverage Map

| FR | Epic | Description |
|----|------|-------------|
| FR1-3, FR14, FR15 | Epic 1 | Initial Setup & Data Discovery |
| FR4-6 | Epic 2 | Backup Continu & Synchronisation |
| FR7-8 | Epic 3 | Validation & Intégrité Données |
| FR9-10 | Epic 4 | State Management & Locking |
| FR11-13 | Epic 5 | Observability & Alerting |
| FR15-16 | Epic 6 | Troubleshooting & CLI Tools |

## Epic List

### Epic 1: Initial Setup & Data Discovery

Permettre à l'opérateur de configurer Data-Guard et découvrir l'état des données.

**User Outcome:** L'opérateur peut déployer Data-Guard sur une application et vérifier l'état initial des données.

**FRs covered:** FR1, FR2, FR3, FR14, FR15

### Epic 2: Backup Continu & Synchronisation

Assurer la sauvegarde continue des données vers S3.

**User Outcome:** Les données de l'app sont automatiquement sauvegardées en continu vers S3.

**FRs covered:** FR4, FR5, FR6

### Epic 3: Validation & Intégrité Données

Valider l'intégrité des données avant et après les opérations de backup/restore.

**User Outcome:** Les données sont vérifiées pour l'intégrité et la validité avant et après opérations.

**FRs covered:** FR7, FR8

### Epic 4: State Management & Locking

Gérer l'état distribué et éviter les conflits entre replicas.

**User Outcome:** Pas de split-brain entre replicas, locks gérés correctement avec TTL et steal.

**FRs covered:** FR9, FR10

### Epic 5: Observability & Alerting

Exposer métriques Prometheus et alertes pour monitoring.

**User Outcome:** L'opérateur peut monitorer l'état et être alerté des problèmes.

**FRs covered:** FR11, FR12, FR13

### Epic 6: Troubleshooting & CLI Tools

Outils de diagnostic pour résoudre les problèmes manuellement.

**User Outcome:** L'opérateur peut diagnostiquer et résoudre les problèmes via CLI.

**FRs covered:** FR15, FR16

## Epic 1: Initial Setup & Data Discovery

**Goal:** Permettre à l'opérateur de configurer Data-Guard et découvrir l'état des données.

### Story 1.1: Configurer Data-Guard via Annotations K8s

**As a** Cluster Operator,
**I want** ajouter les annotations Data-Guard sur un Deployment Kubernetes,
**So that** Data-Guard peut être déployé automatiquement sur mon application.

**Acceptance Criteria:**

**Given** un Deployment Kubernetes pour Home Assistant,
**When** j'ajoute les annotations `data-guard/bucket`, `data-guard/path`, `data-guard/backup-interval`,
**Then** Data-Guard init container et sidecars sont automatiquement ajoutés au pod,
**And** les configurations sont correctement passées aux composants.

### Story 1.2: Init Container Detect Healthy Data

**As a** Cluster Operator,
**I want** l'init container puisse vérifier l'état local vs S3,
**So that** il décide si un restore est nécessaire.

**Acceptance Criteria:**

**Given** des données locales présentes sur le volume,
**When** l'init container démarre et consulte S3,
**Then** il compare la version locale avec la version S3,
**And** il détermine si les données sont valides, corrompues, ou manquantes.

### Story 1.3: Restore Conditionnel ou Skip

**As a** Cluster Operator,
**I want** l'init container puisse restaurer ou skip le restore,
**So that** le pod démarre rapidement si les données sont valides.

**Acceptance Criteria:**

**Given** l'init container a déterminé l'état des données,
**When** les données sont valides et plus récentes que S3,
**Then** l'init container skip le restore et continue,
**And** le pod démarre en < 30s.

### Story 1.4: CLI Verify Backup State

**As a** Cluster Operator,
**I want** un CLI tool pour vérifier l'état du backup S3,
**So that** je peux diagnostiquer les problèmes manuellement.

**Acceptance Criteria:**

**Given** j'accède au CLI depuis mon poste de travail,
**When** j'exécute `dataangel-cli verify --bucket myapp`,
**Then** je vois l'état actuel des backups dans S3,
**And** je vois si des restaurations sont nécessaires.

## Epic 2: Backup Continu & Synchronisation

**Goal:** Assurer la sauvegarde continue des données vers S3.

### Story 2.1: Sidecar Litestream Backup SQLite

**As a** Cluster Operator,
**I want** le sidecar Litestream backup SQLite en continu vers S3,
**So that** mes données sont sauvegardées en temps réel.

**Acceptance Criteria:**

**Given** une application avec base SQLite,
**When** le sidecar Litestream démarre,
**Then** il stream les modifications SQLite vers S3 en continu,
**And** les données sont accessibles pour restauration.

### Story 2.2: Sidecar Rclone Sync Filesystem

**As a** Cluster Operator,
**I want** le sidecar Rclone sync le filesystem vers S3 toutes les 60s,
**So that** les fichiers YAML et autres sont sauvegardés périodiquement.

**Acceptance Criteria:**

**Given** des fichiers YAML dans le répertoire de configuration,
**When** le sidecar Rclone démarre,
**Then** il sync les fichiers vers S3 toutes les 60s,
**And** les fichiers sont accessibles pour restauration.

### Story 2.3: Graceful Shutdown with WAL Flush

**As a** Cluster Operator,
**I want** l'application puisse s'arrêter proprement avec flush du WAL,
**So that** toutes les écritures sont sauvegardées avant l'arrêt.

**Acceptance Criteria:**

**Given** l'application en cours d'exécution,
**When** une terminaison est demandée,
**Then** le sidecar Litestream flush le WAL SQLite,
**And** toutes les écritures sont sauvegardées sur S3.

## Epic 3: Validation & Intégrité Données

**Goal:** Valider l'intégrité des données avant et après les opérations.

### Story 3.1: Pre-Backup Validation SQLite/YAML

**As a** Cluster Operator,
**I want** valider l'intégrité des données avant le backup,
**So that** je ne sauvegarde pas des données corrompues.

**Acceptance Criteria:**

**Given** des données à sauvegarder,
**When** le backup est déclenché,
**Then** la validation SQLite/VAML est exécutée,
**And** le backup est bloqué si les données sont corrompues.

### Story 3.2: Post-Restore Validation

**As a** Cluster Operator,
**I want** valider l'intégrité des données après la restauration,
**So that** je confirme que les données restaurées sont valides.

**Acceptance Criteria:**

**Given** une restauration vient d'être effectuée,
**When** l'init container vérifie les données,
**Then** une validation d'intégrité est exécutée,
**And** une alerte est déclenchée si les données sont invalides.

## Epic 4: State Management & Locking

**Goal:** Gérer l'état distribué et éviter les conflits entre replicas.

### Story 4.1: S3 Distributed Lock Implementation

**As a** Cluster Operator,
**I want** un mécanisme de lock S3 pour éviter le split-brain,
**So that** plusieurs replicas n'écrivent pas simultanément.

**Acceptance Criteria:**

**Given** plusieurs replicas de l'application,
**When** un replica tente d'écrire sur S3,
**Then** il acquiert un lock S3 avant d'écrire,
**And** les autres replicas sont bloqués jusqu'au release du lock.

### Story 4.2: Lock TTL + Steal Mechanism

**As a** Cluster Operator,
**I want** un TTL sur les locks S3 avec mécanisme de steal,
**So that** un lock bloqué peut être libéré automatiquement.

**Acceptance Criteria:**

**Given** un lock S3 a été acquis,
**When** le TTL expire sans release,
**Then** un autre replica peut voler le lock,
**And** le lock est libéré pour le nouveau propriétaire.

## Epic 5: Observability & Alerting

**Goal:** Exposer métriques Prometheus et alertes pour monitoring.

### Story 5.1: Prometheus Metrics Exporter

**As a** Cluster Operator,
**I want** exposer des métriques Prometheus sur les backups/restores,
**So that** je peux monitorer l'état du système.

**Acceptance Criteria:**

**Given** l'application tourne,
**When** Prometheus scrape les métriques,
**Then** les métriques dataguard_backup_duration_seconds sont exposées,
**And** les métriques dataguard_restore_operations_total sont exposées.

### Story 5.2: Alerting Backup Failure

**As a** Cluster Operator,
**I want** recevoir une alerte si le backup échoue,
**So that** je peux intervenir rapidement.

**Acceptance Criteria:**

**Given** un backup échoue (S3 indisponible, etc.),
**When** l'alerting est configuré,
**Then** une alerte est envoyée,
**And** la métrique dataguard_backup_failures_total est incrémentée.

### Story 5.3: Alerting Restore Performed

**As a** Cluster Operator,
**I want** être notifié quand une restauration est effectuée,
**So that** je peux vérifier que le restore s'est bien passé.

**Acceptance Criteria:**

**Given** une restauration a été effectuée,
**When** l'alerting est configuré,
**Then** une notification est envoyée,
**And** la métrique dataguard_restore_operations_total est incrémentée.

## Epic 6: Troubleshooting & CLI Tools

**Goal:** Outils de diagnostic pour résoudre les problèmes manuellement.

### Story 6.1: CLI Verify Backup State

**As a** Cluster Operator,
**I want** un CLI tool pour vérifier l'état des backups S3,
**So that** je peux diagnostiquer les problèmes manuellement.

**Acceptance Criteria:**

**Given** j'accède au CLI depuis mon poste de travail,
**When** j'exécute `dataangel-cli verify --bucket myapp`,
**Then** je vois l'état actuel des backups dans S3,
**And** je vois si des restaurations sont nécessaires.

### Story 6.2: CLI Force Release Lock

**As a** Cluster Operator,
**I want** un CLI tool pour forcer le release d'un lock bloqué,
**So that** je peux résoudre les problèmes de lock.

**Acceptance Criteria:**

**Given** un lock S3 est bloqué,
**When** j'exécute `dataangel-cli force-release-lock --bucket myapp`,
**Then** le lock est libéré immédiatement,
**And** le système peut reprendre son fonctionnement.