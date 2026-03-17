# Déploiement dataAngel dans Kubernetes

## 📦 Image Docker

```
charchess/dataangel:latest
```

L'image est disponible sur Docker Hub et contient:
- `./init` - Init container pour restore au démarrage
- `./cli` - Outils CLI (verify, force-release-lock)

## 🔧 Configuration Kubernetes

### 1. Secret AWS (pour MinIO/S3)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: data-guard-credentials
  namespace: default
type: Opaque
stringData:
  access-key: your-access-key
  secret-key: your-secret-key
```

### 2. Déployment avec Init Container dataAngel

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-application
  namespace: default
  annotations:
    data-guard.io/enabled: "true"
    data-guard.io/bucket: "my-bucket"
    data-guard.io/path: "backups/app.db"
    data-guard.io/checksum: "sha256:abc123..."
    data-guard.io/s3-endpoint: "http://minio.minio.svc.cluster.local:9000"
spec:
  replicas: 1
  selector:
    matchLabels:
      app: my-application
  template:
    metadata:
      labels:
        app: my-application
    spec:
      initContainers:
      - name: data-guard
        image: charchess/dataangel:latest
        command: ["./init"]
        env:
        - name: DATA_GUARD_BUCKET
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/bucket']
        - name: DATA_GUARD_PATH
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/path']
        - name: DATA_GUARD_LOCAL_PATH
          value: "/data/app.db"
        - name: DATA_GUARD_CHECKSUM
          valueFrom:
            fieldRef:
              fieldPath: metadata.annotations['data-guard.io/checksum']
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
        volumeMounts:
        - name: data
          mountPath: /data
      
      containers:
      - name: my-app
        image: my-app:latest
        volumeMounts:
        - name: data
          mountPath: /data
      
      volumes:
      - name: data
        emptyDir: {}
```

### Annotations disponibles

| Annotation | Description | Required |
|------------|-------------|----------|
| `data-guard.io/enabled` | Activer dataGuard | Yes |
| `data-guard.io/bucket` | Nom du bucket S3 | Yes |
| `data-guard.io/path` | Chemin dans le bucket | Yes |
| `data-guard.io/checksum` | Checksum SHA256 attendu | No |
| `data-guard.io/s3-endpoint` | URL S3 (pour MinIO) | No |
| `data-guard.io/full-logs` | Logs détaillés | No |

## 🚀 Déploiement

```bash
# Créer le secret
kubectl create secret generic data-guard-credentials \
  --from-literal=access-key=minioadmin \
  --from-literal=secret-key=minioadmin

# Déployer
kubectl apply -f deployment.yaml

# Vérifier les logs
kubectl logs <pod> -c data-guard
```

## 🔍 CLI

```bash
# Vérifier les backups
kubectl run data-guard --image=charchess/dataangel:latest --rm -i -- ./cli verify --bucket my-bucket

# Forcer la release d'un lock
kubectl run data-guard --image=charchess/dataangel:latest --rm -i -- ./cli force-release-lock --lock-id my-lock
```

## ✅ Checklist

- [ ] Secret credentials créé
- [ ] ImagePullPolicy: Always (pour mettre à jour)
- [ ] Volume mount configuré
- [ ] Checksum défini (recommandé)
