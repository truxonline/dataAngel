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
kubectl create secret generic data-guard-credentials \
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
        data-guard.io/bucket: "dataangel-test"
        data-guard.io/sqlite-paths: "/data/test.db"
        data-guard.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
    spec:
      initContainers:
      - name: data-guard-init
        image: charchess/dataangel:latest
        command: ["./init"]
        env:
        - name: DATA_GUARD_BUCKET
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/bucket']
        - name: DATA_GUARD_SQLITE_PATHS
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/sqlite-paths']
        - name: DATA_GUARD_S3_ENDPOINT
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/s3-endpoint']
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: data-guard-credentials
              key: access-key
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: data-guard-credentials
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
        data-guard.io/bucket: "dataangel-test"
        data-guard.io/fs-paths: "/config"
        data-guard.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
    spec:
      initContainers:
      - name: data-guard-init
        image: charchess/dataangel:latest
        command: ["./init"]
        env:
        - name: DATA_GUARD_BUCKET
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/bucket']
        - name: DATA_GUARD_FS_PATHS
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/fs-paths']
        - name: DATA_GUARD_S3_ENDPOINT
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/s3-endpoint']
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: data-guard-credentials
              key: access-key
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: data-guard-credentials
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
        data-guard.io/bucket: "dataangel-test"
        data-guard.io/sqlite-paths: "/data/test.db"
        data-guard.io/fs-paths: "/config"
        data-guard.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
    spec:
      initContainers:
      - name: data-guard-init
        image: charchess/dataangel:latest
        command: ["./init"]
        env:
        - name: DATA_GUARD_BUCKET
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/bucket']
        - name: DATA_GUARD_SQLITE_PATHS
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/sqlite-paths']
        - name: DATA_GUARD_FS_PATHS
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/fs-paths']
        - name: DATA_GUARD_S3_ENDPOINT
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/s3-endpoint']
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: data-guard-credentials
              key: access-key
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: data-guard-credentials
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

## Checklist Final

Une fois tous les tests passés:

- [ ] Test 1 (SQLite-only): ✅
- [ ] Test 2 (FS-only): ✅
- [ ] Test 3 (Combined): ✅
- [ ] Test 4 (Skip behavior): ✅
- [ ] Test 5 (Error handling): ✅
- [ ] Métriques Prometheus accessibles (port 9090)
- [ ] Sidecar continuous backup fonctionne
- [ ] Documentation à jour (README.md, DEPLOYMENT.md)

## Cleanup

```bash
kubectl delete deployment test-sqlite-only test-fs-only test-combined
kubectl delete secret data-guard-credentials
mc rb --force minio/dataangel-test
```
