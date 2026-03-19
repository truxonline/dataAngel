# Manuel de Test dataAngel

Ce document décrit comment tester les 3 modes de restore après implémentation du devops_brief.

## Prérequis

- Cluster K8s (vixens) accessible
- MinIO déployé sur `http://minio.minio.svc.cluster.local:9000`
- Credentials MinIO (access-key / secret-key)
- kubectl configuré
- Docker image `charchess/dataangel:latest` rebuilt (après commit 53d4370)

## Setup Initial

### 1. Créer le secret AWS

```bash
kubectl create secret generic dataangel-credentials \
  --from-literal=access-key=YOUR_MINIO_ACCESS_KEY \
  --from-literal=secret-key=YOUR_MINIO_SECRET_KEY \
  -n default
```

### 2. Créer un bucket MinIO pour tests

```bash
# Via MinIO Console ou mc CLI
mc mb minio/dataangel-test
```

### 3. Peupler le bucket avec des données de test

#### SQLite test data

```bash
# Créer une DB SQLite locale
sqlite3 test.db "CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT); INSERT INTO users VALUES (1, 'Alice');"

# Utiliser litestream pour pousser vers MinIO
export AWS_ACCESS_KEY_ID=YOUR_KEY
export AWS_SECRET_ACCESS_KEY=YOUR_SECRET
export AWS_ENDPOINT_URL=http://minio.minio.svc.cluster.local:9000

litestream replicate test.db s3://dataangel-test/sqlite/test.db
```

#### Filesystem test data

```bash
# Créer des fichiers test
mkdir -p testdata/config
echo "app_setting=true" > testdata/config/app.conf
echo "hello world" > testdata/config/readme.txt

# Pousser vers MinIO avec rclone
rclone copy testdata/config :s3:dataangel-test/filesystem/config \
  --s3-provider=Minio \
  --s3-endpoint=http://minio.minio.svc.cluster.local:9000 \
  --s3-access-key-id=$AWS_ACCESS_KEY_ID \
  --s3-secret-access-key=$AWS_SECRET_ACCESS_KEY
```

## Test 1: SQLite-only Restore

### Deployment manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-sqlite-only
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-sqlite-only
  template:
    metadata:
      labels:
        app: test-sqlite-only
      annotations:
        dataangel.io/bucket: "dataangel-test"
        dataangel.io/sqlite-paths: "/data/test.db"
        dataangel.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
    spec:
      initContainers:
      - name: data-guard-init
        image: charchess/dataangel:latest
        command: ["./init"]
        env:
        - name: DATA_GUARD_BUCKET
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/bucket']
        - name: DATA_GUARD_SQLITE_PATHS
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/sqlite-paths']
        - name: DATA_GUARD_S3_ENDPOINT
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/s3-endpoint']
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: dataangel-credentials
              key: access-key
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: dataangel-credentials
              key: secret-key
        - name: DATA_GUARD_FULL_LOGS
          value: "true"
        volumeMounts:
        - name: data
          mountPath: /data
      
      containers:
      - name: alpine
        image: alpine:latest
        command: ["sleep", "infinity"]
        volumeMounts:
        - name: data
          mountPath: /data
      
      volumes:
      - name: data
        emptyDir: {}
```

### Validation

```bash
# Déployer
kubectl apply -f test-sqlite-only.yaml

# Attendre que le pod soit Running
kubectl wait --for=condition=Ready pod -l app=test-sqlite-only --timeout=60s

# Vérifier les logs de l'init container
kubectl logs -l app=test-sqlite-only -c data-guard-init

# Expected output:
# [INFO] Restoring SQLite: /data/test.db
# [INFO] Running: litestream restore -if-db-not-exists -if-replica-exists ...
# [INFO] SQLite restore completed: /data/test.db

# Vérifier que la DB existe et contient les données
kubectl exec -it deploy/test-sqlite-only -- sh
# Dans le pod:
ls -lh /data/test.db
# Devrait montrer un fichier test.db

# Vérifier le contenu avec sqlite3 (si disponible dans alpine)
# Sinon, copier localement:
exit
kubectl cp test-sqlite-only-xxx:/data/test.db ./restored-test.db
sqlite3 restored-test.db "SELECT * FROM users;"
# Expected: 1|Alice
```

### Success Criteria

- ✅ Init container exit code 0
- ✅ Logs montrent "litestream restore" command executed
- ✅ Fichier `/data/test.db` existe dans le pod
- ✅ DB contient les données originales (table users avec Alice)

## Test 2: Filesystem-only Restore

### Deployment manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-fs-only
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-fs-only
  template:
    metadata:
      labels:
        app: test-fs-only
      annotations:
        dataangel.io/bucket: "dataangel-test"
        dataangel.io/fs-paths: "/config"
        dataangel.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
    spec:
      initContainers:
      - name: data-guard-init
        image: charchess/dataangel:latest
        command: ["./init"]
        env:
        - name: DATA_GUARD_BUCKET
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/bucket']
        - name: DATA_GUARD_FS_PATHS
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/fs-paths']
        - name: DATA_GUARD_S3_ENDPOINT
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/s3-endpoint']
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: dataangel-credentials
              key: access-key
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: dataangel-credentials
              key: secret-key
        - name: DATA_GUARD_FULL_LOGS
          value: "true"
        volumeMounts:
        - name: config
          mountPath: /config
      
      containers:
      - name: alpine
        image: alpine:latest
        command: ["sleep", "infinity"]
        volumeMounts:
        - name: config
          mountPath: /config
      
      volumes:
      - name: config
        emptyDir: {}
```

### Validation

```bash
kubectl apply -f test-fs-only.yaml
kubectl wait --for=condition=Ready pod -l app=test-fs-only --timeout=60s
kubectl logs -l app=test-fs-only -c data-guard-init

# Expected output:
# [INFO] Restoring filesystem: /config
# [INFO] Running: rclone copy :s3:dataangel-test/filesystem/config /config --exclude "*.db*"
# [INFO] Filesystem restore completed: /config

# Vérifier les fichiers
kubectl exec -it deploy/test-fs-only -- ls -lh /config
# Expected: app.conf, readme.txt

kubectl exec -it deploy/test-fs-only -- cat /config/app.conf
# Expected: app_setting=true
```

### Success Criteria

- ✅ Init container exit code 0
- ✅ Logs montrent "rclone copy" command executed
- ✅ Fichiers `/config/app.conf` et `/config/readme.txt` existent
- ✅ Contenu des fichiers correspond aux originaux
- ✅ Aucun fichier `.db` copié (grâce à `--exclude "*.db*"`)

## Test 3: Combined Mode (SQLite + Filesystem)

### Deployment manifest

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-combined
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: test-combined
  template:
    metadata:
      labels:
        app: test-combined
      annotations:
        dataangel.io/bucket: "dataangel-test"
        dataangel.io/sqlite-paths: "/data/test.db"
        dataangel.io/fs-paths: "/config"
        dataangel.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
    spec:
      initContainers:
      - name: data-guard-init
        image: charchess/dataangel:latest
        command: ["./init"]
        env:
        - name: DATA_GUARD_BUCKET
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/bucket']
        - name: DATA_GUARD_SQLITE_PATHS
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/sqlite-paths']
        - name: DATA_GUARD_FS_PATHS
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/fs-paths']
        - name: DATA_GUARD_S3_ENDPOINT
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/s3-endpoint']
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: dataangel-credentials
              key: access-key
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: dataangel-credentials
              key: secret-key
        - name: DATA_GUARD_FULL_LOGS
          value: "true"
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /config
      
      containers:
      - name: alpine
        image: alpine:latest
        command: ["sleep", "infinity"]
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /config
      
      volumes:
      - name: data
        emptyDir: {}
      - name: config
        emptyDir: {}
```

### Validation

```bash
kubectl apply -f test-combined.yaml
kubectl wait --for=condition=Ready pod -l app=test-combined --timeout=60s
kubectl logs -l app=test-combined -c data-guard-init

# Expected output:
# [INFO] Restoring SQLite: /data/test.db
# [INFO] Running: litestream restore ...
# [INFO] SQLite restore completed: /data/test.db
# [INFO] Restoring filesystem: /config
# [INFO] Running: rclone copy ...
# [INFO] Filesystem restore completed: /config

# Vérifier BOTH SQLite et filesystem
kubectl exec -it deploy/test-combined -- sh
ls -lh /data/test.db /config/app.conf
# Les deux doivent exister
exit
```

### Success Criteria

- ✅ Init container exit code 0
- ✅ Logs montrent BOTH litestream ET rclone commands
- ✅ SQLite DB `/data/test.db` existe avec données
- ✅ Fichiers `/config/*` existent avec contenu correct
- ✅ Les deux opérations se sont exécutées indépendamment

## Test 4: Skip Behavior (DB existe déjà)

Ce test valide que l'init container skip le restore si la DB existe déjà localement.

```bash
# Déployer une fois (première restoration)
kubectl apply -f test-sqlite-only.yaml
kubectl wait --for=condition=Ready pod -l app=test-sqlite-only

# Modifier la DB dans le pod
kubectl exec -it deploy/test-sqlite-only -- sh -c "echo 'modified' > /data/test.db"

# Scale down puis up (force restart)
kubectl scale deployment test-sqlite-only --replicas=0
sleep 2
kubectl scale deployment test-sqlite-only --replicas=1
kubectl wait --for=condition=Ready pod -l app=test-sqlite-only --timeout=60s

# Vérifier les logs
kubectl logs -l app=test-sqlite-only -c data-guard-init

# Expected: litestream devrait skip avec "-if-db-not-exists"
# La DB devrait GARDER "modified" (pas écrasée)
```

### Success Criteria

- ✅ Init container exit code 0 même en skip
- ✅ Logs montrent que litestream a skip (DB exists)
- ✅ DB locale pas écrasée

## Test 5: Error Handling (Credentials invalides)

```yaml
# Modifier le deployment pour utiliser de mauvais credentials
env:
- name: AWS_ACCESS_KEY_ID
  value: "INVALID_KEY"
- name: AWS_SECRET_ACCESS_KEY
  value: "INVALID_SECRET"
```

```bash
kubectl apply -f test-error.yaml

# Attendre que le pod fail
kubectl get pods -l app=test-error -w

# Vérifier les logs d'erreur
kubectl logs -l app=test-error -c data-guard-init

# Expected: Exit code non-zero, error logs S3 auth failed
```

### Success Criteria

- ✅ Init container exit code non-zero
- ✅ Logs montrent erreur S3 authentification
- ✅ Pod reste en Init:Error state

## Test 6: Corruption Detection (Issue #3)

Ce test valide la détection et récupération automatique des bases SQLite corrompues.

### Contexte

Si une DB locale existe mais est corrompue (header écrasé, fichier tronqué), l'init container doit :
1. Détecter la corruption via `PRAGMA integrity_check`
2. Supprimer la DB corrompue
3. Restaurer depuis S3

**Sans le fix** (issue #3) : Litestream skip (`-if-db-not-exists`), app crash avec DB corrompue.

### Validation automatique (tests unitaires)

```bash
# Dans le repository local
cd cmd/init
go test -v -run TestIsSQLiteHealthy

# Expected output:
# === RUN   TestIsSQLiteHealthy/healthy_database        ✅
# === RUN   TestIsSQLiteHealthy/corrupted_database_header  ✅
# === RUN   TestIsSQLiteHealthy/non-existent_database   ✅
# === RUN   TestIsSQLiteHealthy/empty_file              ✅
# PASS
```

### Validation manuelle (cluster K8s)

Simuler une corruption et vérifier la récupération automatique :

```bash
# 1. Déployer avec DB saine
kubectl apply -f test-sqlite-only.yaml
kubectl wait --for=condition=Ready pod -l app=test-sqlite-only

# 2. Corrompre la DB manuellement
kubectl exec -it deploy/test-sqlite-only -- sh -c "dd if=/dev/urandom of=/data/test.db bs=512 count=8 conv=notrunc"

# 3. Redémarrer le pod (trigger init container)
kubectl delete pod -l app=test-sqlite-only
kubectl wait --for=condition=Ready pod -l app=test-sqlite-only

# 4. Vérifier les logs init
kubectl logs -l app=test-sqlite-only -c data-guard-init
```

**Expected logs**:
```
WARNING: Database exists but is corrupted, removing: /data/test.db
Corrupted database removed, proceeding with restore
Running: litestream restore -config /tmp/litestream-restore-*.yml -if-db-not-exists -if-replica-exists /data/test.db
SQLite restored successfully: /data/test.db
```

**NOT expected** (old behavior):
```
database already exists, skipping
SQLite restored successfully
```

### Cas testés

| Cas DB locale | Backup S3 | Comportement attendu |
|---------------|-----------|---------------------|
| Pas de DB | Présent | ✅ Restore depuis S3 |
| DB saine | Présent | ✅ Skip restore (DB existe) |
| DB corrompue (header écrasé) | Présent | ✅ **Supprimer + Restore** |
| DB vide (0 bytes) | Présent | ✅ **Supprimer + Restore** |
| DB corrompue | Pas de backup | ⚠️ Supprimer, restore fail (no replica) |

### Success Criteria

- ✅ `isSQLiteHealthy()` détecte header corrompu
- ✅ `isSQLiteHealthy()` détecte fichier vide
- ✅ `isSQLiteHealthy()` valide DB saine (retourne true)
- ✅ DB corrompue supprimée automatiquement
- ✅ Restore S3 se déclenche après suppression
- ✅ Logs montrent "WARNING: Database exists but is corrupted"
- ✅ App démarre avec DB restaurée (pas de crash)

### Vérifier l'intégrité après restore

```bash
kubectl exec -it deploy/test-sqlite-only -- sh
# Dans le pod:
sqlite3 /data/test.db "PRAGMA integrity_check;"
# Expected: ok
```

## Test 7: Rclone Timeout on Empty S3 Prefix (Issue #4)

Ce test valide que rclone n'hang pas indéfiniment quand le prefix S3 n'existe pas.

### Contexte

Au premier démarrage (bucket vide, pas d'historique FS), `rclone copy` pouvait hang indéfiniment sans retourner.

**Fix appliqué** : Ajout de timeouts rclone :
- `--timeout 60s` : timeout global opération
- `--contimeout 15s` : timeout connexion S3

### Validation

```bash
# 1. Bucket vide (pas de prefix filesystem)
# Pas de setup préalable nécessaire

# 2. Déployer avec FS_PATHS
kubectl apply -f test-fs-only.yaml
# annotations:
#   dataangel.io/fs-paths: "/config"

# 3. Attendre max 60s (timeout)
kubectl wait --for=condition=Ready pod -l app=test-fs-only --timeout=90s
```

**Expected behavior** :
- Init container termine en <90s (même avec prefix vide)
- Logs : `rclone copy` se termine (exit 0 ou timeout)
- Pod devient Ready

**NOT expected** (old behavior) :
- Init hang >2min
- Pod jamais Ready

### Success Criteria

- ✅ Init container termine en <90s
- ✅ Pas de hang indéfini
- ✅ Pod devient Ready même sans backup FS préexistant
- ✅ Sidecar rclone sync timeout configuré (120s)

## Test 8: Metrics Instrumentation (Issue #5)

Ce test valide que les métriques Prometheus reflètent l'état réel des subprocess.

### Contexte

Les métriques étaient exposées mais toujours à 0, même en cas d'erreur.

**Fix appliqué** :
- Instrumentation rclone : `SyncsTotal`, `SyncsFailed`, `RcloneUp`, `SyncDuration`
- Instrumentation litestream : `LitestreamUp`
- Métriques updated après chaque cycle

### Validation

```bash
# 1. Déployer avec métriques activées
kubectl apply -k kustomize/examples/mealie/
# annotation: dataangel.io/metrics-enabled: "true"

# 2. Attendre quelques syncs rclone (~2min)
sleep 120

# 3. Scraper les métriques
kubectl port-forward -n mealie deploy/mealie 9090:9090 &
curl http://localhost:9090/metrics | grep dataguard
```

**Expected metrics (après 2 syncs réussis)** :
```
dataguard_litestream_up 1
dataguard_rclone_up 1
dataguard_rclone_syncs_total 2
dataguard_rclone_syncs_failed_total 0
dataguard_rclone_sync_duration_seconds_bucket{le="+Inf"} 2
dataguard_sidecar_uptime_seconds 120
```

### Test erreur S3

Simuler une panne S3 et vérifier que les métriques reflètent l'état :

```bash
# 1. Bloquer accès MinIO (firewall ou scale down)
kubectl scale -n minio deployment minio --replicas=0

# 2. Attendre un cycle rclone (60s)
sleep 70

# 3. Vérifier métriques
curl http://localhost:9090/metrics | grep dataguard_rclone
```

**Expected (S3 down)** :
```
dataguard_rclone_up 0
dataguard_rclone_syncs_failed_total 1
```

**Expected logs sidecar** :
```
rclone sync error: ...
```

### Success Criteria

- ✅ `rclone_syncs_total` incrémente après chaque sync réussi
- ✅ `rclone_syncs_failed_total` incrémente en cas d'erreur
- ✅ `rclone_up` = 1 quand OK, 0 quand erreur
- ✅ `litestream_up` = 1 au démarrage
- ✅ `sync_duration_seconds` enregistre les durées réelles
- ✅ Métriques utilisables pour alerting Prometheus

## Test 9: MinIO Endpoint Validation (Issue #1)

Ce test valide le fix pour [issue #1](https://github.com/truxonline/dataAngel/issues/1): litestream restore avec endpoint custom.

### Contexte

Avant le fix, l'init container utilisait le flag `-endpoint` qui n'existe pas dans `litestream restore`, causant:
```
flag provided but not defined: -endpoint
```

Après le fix, un fichier de config temporaire est généré avec le field `endpoint:` dans le YAML.

### Validation automatique (tests unitaires)

```bash
# Dans le repository local
cd cmd/init
go test -v -run TestGenerateLitestreamConfig

# Expected output:
# === RUN   TestGenerateLitestreamConfig
# === RUN   TestGenerateLitestreamConfig/with_custom_endpoint
# === RUN   TestGenerateLitestreamConfig/without_custom_endpoint
# --- PASS: TestGenerateLitestreamConfig (0.00s)
```

### Validation manuelle (cluster K8s)

Déployer avec endpoint MinIO et vérifier les logs:

```bash
# Déployer test-sqlite-only (voir Test 1)
kubectl apply -f test-sqlite-only.yaml

# Attendre pod ready
kubectl wait --for=condition=Ready pod -l app=test-sqlite-only

# Vérifier logs init container
kubectl logs -l app=test-sqlite-only -c data-guard-init
```

**Expected logs**:
```
Running: litestream restore -config /tmp/litestream-restore-<pid>.yml -if-db-not-exists -if-replica-exists /data/test.db
SQLite restored successfully: /data/test.db
```

**NOT expected** (old behavior):
```
Running: litestream restore ... -endpoint http://minio:9000 ...
flag provided but not defined: -endpoint
```

### Vérifier le fichier config généré

Si besoin de debug, inspecter le config temporaire:

```bash
# Modifier cmd/init/restore.go temporairement pour ne pas supprimer le config
# Ligne: defer os.Remove(configPath)  # Commenter cette ligne

# Rebuild image et redéployer
# Puis exec dans le pod avant qu'il termine

kubectl exec -it <pod> -c data-guard-init -- cat /tmp/litestream-restore-*.yml
```

**Expected output**:
```yaml
dbs:
  - path: /data/test.db
    replicas:
      - url: s3://dataangel-test/test.db
        endpoint: http://minio.minio.svc.cluster.local:9000
```

### Success Criteria

- ✅ Init container démarre sans erreur "flag provided but not defined"
- ✅ Logs montrent `-config /tmp/litestream-restore-*.yml` (pas `-endpoint`)
- ✅ DB restaurée correctement depuis MinIO
- ✅ Tests unitaires passent (`TestGenerateLitestreamConfig`)

### Régression à surveiller

Vérifier que le mode **sans** endpoint custom fonctionne toujours:

```yaml
# Sans dataangel.io/s3-endpoint annotation
annotations:
  dataangel.io/bucket: "my-bucket"
  dataangel.io/sqlite-paths: "/data/test.db"
  # Pas de s3-endpoint → utilise AWS S3 standard
```

**Expected logs**:
```
Running: litestream restore -if-db-not-exists -if-replica-exists -replica s3://my-bucket/test.db /data/test.db
```

Pas de fichier config généré, utilise le flag `-replica` directement.

## Test 10: Métriques Optionnelles (metrics-enabled annotation)

Ce test valide que le serveur de métriques démarre/skip selon l'annotation.

### Test 10a: Métriques activées (production)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-metrics-enabled
  annotations:
    dataangel.io/bucket: "dataangel-test"
    dataangel.io/sqlite-paths: "/data/test.db"
    dataangel.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
    dataangel.io/metrics-enabled: "true"  # Activer métriques
spec:
  # ... (spec identique à test-sqlite-only)
```

**Validation**:
```bash
kubectl apply -f test-metrics-enabled.yaml
kubectl wait --for=condition=Ready pod -l app=test-metrics-enabled

# Vérifier logs sidecar
kubectl logs -l app=test-metrics-enabled -c data-guard-sidecar | grep "Starting metrics server"
# Expected: "Starting metrics server on :9090"

# Tester endpoint metrics
kubectl port-forward -l app=test-metrics-enabled 9090:9090 &
curl http://localhost:9090/metrics | grep dataguard
# Expected: métriques exposées (dataguard_litestream_up, dataguard_rclone_up, etc.)
```

**Success Criteria**:
- ✅ Logs montrent "Starting metrics server on :9090"
- ✅ Port 9090 accessible et répond avec métriques Prometheus
- ✅ Métriques `dataguard_*` présentes

### Test 10b: Métriques désactivées (dev/CI)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-metrics-disabled
  annotations:
    dataangel.io/bucket: "dataangel-test"
    dataangel.io/sqlite-paths: "/data/test.db"
    dataangel.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
    dataangel.io/metrics-enabled: "false"  # Désactiver métriques
spec:
  # ... (spec identique)
```

**Validation**:
```bash
kubectl apply -f test-metrics-disabled.yaml
kubectl wait --for=condition=Ready pod -l app=test-metrics-disabled

# Vérifier logs sidecar
kubectl logs -l app=test-metrics-disabled -c data-guard-sidecar | grep metrics
# Expected: "Metrics server disabled (DATA_GUARD_METRICS_ENABLED=false)"

# Tester que le port 9090 ne répond PAS
kubectl port-forward -l app=test-metrics-disabled 9090:9090 &
curl http://localhost:9090/metrics
# Expected: Connection refused ou timeout
```

**Success Criteria**:
- ✅ Logs montrent "Metrics server disabled"
- ✅ Port 9090 ne répond pas (serveur pas démarré)
- ✅ Sidecar continue de fonctionner normalement (Litestream + Rclone actifs)

### Test 10c: PodMonitor discovery (avec Prometheus Operator)

**Prérequis**: Prometheus Operator installé (CRD `monitoring.coreos.com/v1`)

```yaml
# kustomization.yaml
components:
  - ../../components/data-guard
  - ../../components/dataangel-monitoring  # PodMonitor pour découverte
```

**Validation**:
```bash
# Déployer avec component monitoring
kubectl apply -k kustomize/examples/mealie/

# Vérifier que le PodMonitor est créé
kubectl get podmonitor data-guard-sidecar -n mealie
# Expected: PodMonitor exists

# Vérifier labels pour discovery
kubectl get podmonitor data-guard-sidecar -n mealie -o yaml | grep -A5 labels
# Expected: release: prometheus, app.kubernetes.io/name: data-guard

# Vérifier que Prometheus découvre le target
# (dans l'UI Prometheus: Status > Targets)
# Expected: Target "data-guard-sidecar" avec état UP
```

**Success Criteria**:
- ✅ PodMonitor créé avec labels corrects (`release: prometheus`)
- ✅ Prometheus découvre le target automatiquement
- ✅ Métriques `dataguard_*` visibles dans Prometheus UI

## Checklist Final

Une fois tous les tests passés:

- [ ] Test 1 (SQLite-only): ✅
- [ ] Test 2 (FS-only): ✅
- [ ] Test 3 (Combined): ✅
- [ ] Test 4 (Skip behavior): ✅
- [ ] Test 5 (Error handling): ✅
- [ ] Test 6 (Corruption detection): ✅ (fix issue #3)
- [ ] Test 7 (Rclone timeout): ✅ (fix issue #4)
- [ ] Test 8 (Metrics instrumentation): ✅ (fix issue #5)
- [ ] Test 9 (MinIO endpoint): ✅ (fix issue #1)
- [ ] Test 10a (Métriques activées): ✅
- [ ] Test 10b (Métriques désactivées): ✅
- [ ] Test 10c (PodMonitor discovery): ✅ (si Prometheus Operator installé)
- [ ] Sidecar continuous backup fonctionne
- [ ] Documentation à jour (README.md, devops_brief.md)

## Cleanup

```bash
kubectl delete deployment test-sqlite-only test-fs-only test-combined
kubectl delete secret dataangel-credentials
mc rb --force minio/dataangel-test
```
