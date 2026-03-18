# data-guard Kustomize Component

Ce component injecte automatiquement l'init container **et** le sidecar data-guard dans vos Deployments.

## Usage de base

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Kustomization

components:
  - ../../components/data-guard

resources:
  - deployment.yaml
```

## Annotations requises

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  annotations:
    data-guard.io/bucket: "my-backup-bucket"
    data-guard.io/sqlite-paths: "/data/app.db"           # Optionnel (si SQLite)
    data-guard.io/fs-paths: "/config"                    # Optionnel (si filesystem)
    data-guard.io/s3-endpoint: "http://minio:9000"       # Optionnel (défaut: AWS S3)
    data-guard.io/rclone-interval: "300s"                # Optionnel (défaut: 60s)
spec:
  template:
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        volumeMounts:
        - name: data
          mountPath: /data
      volumes:
      - name: data
        emptyDir: {}
```

**Note**: Au moins **un** de `sqlite-paths` ou `fs-paths` doit être défini.

## Secret AWS requis

Créez le secret `data-guard-credentials` dans le namespace de votre app:

```bash
kubectl create secret generic data-guard-credentials \
  --from-literal=access-key=YOUR_ACCESS_KEY \
  --from-literal=secret-key=YOUR_SECRET_KEY
```

## Override du mountPath (défaut: /data)

Le component monte par défaut le volume `data` sur `/data`. Si votre app utilise un autre path, créez un patch dans votre overlay:

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Kustomization

components:
  - ../../components/data-guard

resources:
  - deployment.yaml

patches:
  # Override init container mountPath
  - target:
      kind: Deployment
      name: myapp
    patch: |-
      - op: replace
        path: /spec/template/spec/initContainers/0/volumeMounts/0/mountPath
        value: /app/data
  
  # Override sidecar mountPath
  - target:
      kind: Deployment
      name: myapp
    patch: |-
      - op: replace
        path: /spec/template/spec/containers/1/volumeMounts/0/mountPath
        value: /app/data
```

**Attention**: Adapter les index (`initContainers/0`, `containers/1`) selon votre configuration.

## Modes supportés

### SQLite seul (Litestream)
```yaml
annotations:
  data-guard.io/sqlite-paths: "/data/db.sqlite"
```

### Filesystem seul (Rclone)
```yaml
annotations:
  data-guard.io/fs-paths: "/config,/data/uploads"
```

### Les deux ensemble
```yaml
annotations:
  data-guard.io/sqlite-paths: "/data/app.db"
  data-guard.io/fs-paths: "/config"
```

## Métriques Prometheus

Le sidecar expose des métriques sur le port `9090`:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: myapp-metrics
spec:
  selector:
    app: myapp
  ports:
  - name: metrics
    port: 9090
    targetPort: 9090
```
