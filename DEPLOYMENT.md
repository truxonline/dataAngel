# Déploiement dataAngel dans Kubernetes

## 📦 Image Docker

```
charchess/dataangel:latest
```

L'image est disponible sur Docker Hub et contient:
- `./init` - Init container pour restore au démarrage (litestream + rclone)
- `./sidecar` - Sidecar pour backup continu (litestream + rclone)
- `./cli` - Outils CLI (verify, force-release-lock)

## 🔧 Configuration Kubernetes

### 1. Secret AWS (pour MinIO/S3)

Créer le secret **dans le même namespace** que votre application:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: dataangel-credentials
  namespace: default
type: Opaque
stringData:
  access-key: your-access-key
  secret-key: your-secret-key
```

### 2. Modes de déploiement

#### Mode SQLite seul (Litestream)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-application
  namespace: default
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-application
  template:
    metadata:
      labels:
        app: my-application
      annotations:
        dataangel.io/bucket: "my-backup-bucket"
        dataangel.io/sqlite-paths: "/data/app.db"
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
        volumeMounts:
        - name: data
          mountPath: /data
      
      containers:
      - name: my-app
        image: my-app:latest
        volumeMounts:
        - name: data
          mountPath: /data
      
      - name: data-guard-sidecar
        image: charchess/dataangel:latest
        command: ["./sidecar"]
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
        - name: DATA_GUARD_METRICS_PORT
          value: "9090"
        ports:
        - containerPort: 9090
          name: metrics
        volumeMounts:
        - name: data
          mountPath: /data
      
      volumes:
      - name: data
        emptyDir: {}
```

#### Mode Filesystem seul (Rclone)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-application
spec:
  template:
    metadata:
      annotations:
        dataangel.io/bucket: "my-backup-bucket"
        dataangel.io/fs-paths: "/config,/data/uploads"
        dataangel.io/rclone-interval: "300s"
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
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /config
      
      containers:
      - name: my-app
        image: my-app:latest
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /config
      
      - name: data-guard-sidecar
        image: charchess/dataangel:latest
        command: ["./sidecar"]
        env:
        - name: DATA_GUARD_BUCKET
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/bucket']
        - name: DATA_GUARD_FS_PATHS
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/fs-paths']
        - name: DATA_GUARD_RCLONE_INTERVAL
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['dataangel.io/rclone-interval']
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
        - name: DATA_GUARD_METRICS_PORT
          value: "9090"
        ports:
        - containerPort: 9090
          name: metrics
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

#### Mode combiné (SQLite + Filesystem)

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-application
spec:
  template:
    metadata:
      annotations:
        dataangel.io/bucket: "my-backup-bucket"
        dataangel.io/sqlite-paths: "/data/app.db"
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
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /config
      
      containers:
      - name: my-app
        image: my-app:latest
        volumeMounts:
        - name: data
          mountPath: /data
        - name: config
          mountPath: /config
      
      - name: data-guard-sidecar
        image: charchess/dataangel:latest
        command: ["./sidecar"]
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
        - name: DATA_GUARD_METRICS_PORT
          value: "9090"
        ports:
        - containerPort: 9090
          name: metrics
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

### Annotations disponibles

**IMPORTANT**: Les annotations doivent être placées dans `spec.template.metadata.annotations` (pas `metadata.annotations` du Deployment).

| Annotation | Description | Required | Default |
|------------|-------------|----------|---------|
| `dataangel.io/bucket` | Nom du bucket S3 | **Yes** | - |
| `dataangel.io/sqlite-paths` | Chemins SQLite (comma-separated) | No* | - |
| `dataangel.io/fs-paths` | Chemins filesystem (comma-separated) | No* | - |
| `dataangel.io/s3-endpoint` | URL S3 (pour MinIO) | No | AWS S3 |
| `dataangel.io/rclone-interval` | Intervalle rclone sync | No | `60s` |
| `dataangel.io/full-logs` | Logs détaillés | No | `false` |

*Au moins **un** de `sqlite-paths` ou `fs-paths` doit être défini.

## 🚀 Déploiement

```bash
# 1. Créer le secret AWS
kubectl create secret generic dataangel-credentials \
  --from-literal=access-key=minioadmin \
  --from-literal=secret-key=minioadmin \
  -n default

# 2. Déployer votre application
kubectl apply -f deployment.yaml

# 3. Vérifier les logs init container
kubectl logs <pod> -c data-guard-init

# 4. Vérifier les logs sidecar
kubectl logs <pod> -c data-guard-sidecar -f

# 5. Vérifier les métriques Prometheus
kubectl port-forward <pod> 9090:9090
curl http://localhost:9090/metrics
```

## 🔍 CLI

```bash
# Vérifier les backups
kubectl run data-guard --image=charchess/dataangel:latest --rm -i -- \
  ./cli verify --bucket my-bucket

# Forcer la release d'un lock
kubectl run data-guard --image=charchess/dataangel:latest --rm -i -- \
  ./cli force-release-lock --lock-id my-lock
```

## 📊 Métriques Prometheus

Le sidecar expose les métriques suivantes sur le port `9090`:

- `dataguard_litestream_replicas` - Nombre de réplicas Litestream actives
- `dataguard_rclone_syncs_total` - Nombre de syncs rclone effectués
- `dataguard_rclone_sync_errors_total` - Nombre d'erreurs de sync
- `dataguard_rclone_last_sync_timestamp` - Timestamp du dernier sync réussi

Pour scraper avec Prometheus:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: my-app-metrics
  labels:
    app: my-application
spec:
  ports:
  - name: metrics
    port: 9090
    targetPort: 9090
  selector:
    app: my-application
```

Ajouter l'annotation Prometheus sur le Service:
```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "9090"
    prometheus.io/path: "/metrics"
```

## ✅ Checklist

- [ ] Secret `dataangel-credentials` créé dans le namespace
- [ ] Annotations placées dans `spec.template.metadata.annotations`
- [ ] Au moins un de `sqlite-paths` ou `fs-paths` défini
- [ ] Volume mount configuré correctement
- [ ] Sidecar container ajouté pour backup continu
- [ ] Métriques Prometheus scrapées (optionnel)

## 🐛 Troubleshooting

### Init container en erreur

```bash
# Vérifier les logs
kubectl logs <pod> -c data-guard-init

# Erreurs courantes:
# - "missing required env vars" → Annotations mal placées (utiliser spec.template.metadata)
# - "failed to restore" → Vérifier credentials AWS et endpoint S3
# - "database already exists" → Normal, skip automatique si DB existe
```

### Sidecar ne backup pas

```bash
# Vérifier les logs
kubectl logs <pod> -c data-guard-sidecar -f

# Vérifier les métriques
kubectl port-forward <pod> 9090:9090
curl http://localhost:9090/metrics | grep dataguard

# Erreurs courantes:
# - "no SQLite paths configured" → Définir sqlite-paths ou fs-paths
# - "litestream failed to start" → Vérifier format du path SQLite
# - "rclone sync failed" → Vérifier credentials AWS
```

### Credentials AWS invalides

```bash
# Tester manuellement dans un pod debug
kubectl run debug --image=charchess/dataangel:latest --rm -i -- sh

# Dans le shell:
export AWS_ACCESS_KEY_ID=your-key
export AWS_SECRET_ACCESS_KEY=your-secret
export AWS_ENDPOINT_URL=http://minio.minio.svc.cluster.local:9000

# Tester avec AWS CLI
aws s3 ls s3://my-bucket --endpoint-url $AWS_ENDPOINT_URL
```

## 🔐 Sécurité

- **Jamais** commit le secret AWS dans Git
- Utiliser Sealed Secrets ou External Secrets Operator en production
- Limiter les permissions IAM au strict minimum (s3:GetObject, s3:PutObject, s3:ListBucket)
- Activer le chiffrement S3 server-side (SSE-S3 ou SSE-KMS)
