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

⚠️ **Le component utilise par défaut un secret nommé `data-guard-credentials`.**

Créez-le dans le namespace de votre app:

```bash
kubectl create secret generic data-guard-credentials \
  --from-literal=access-key=YOUR_ACCESS_KEY \
  --from-literal=secret-key=YOUR_SECRET_KEY
```

### Override du nom du secret

Si votre app utilise un secret différent (e.g., secret Infisical par app), ajoutez un patch:

```yaml
# kustomization.yaml
patches:
  # Override secret name pour init container
  - target:
      kind: Deployment
      name: myapp
    patch: |-
      - op: replace
        path: /spec/template/spec/initContainers/0/env/3/valueFrom/secretKeyRef/name
        value: myapp-infisical-secret
      - op: replace
        path: /spec/template/spec/initContainers/0/env/4/valueFrom/secretKeyRef/name
        value: myapp-infisical-secret
  
  # Override secret name pour sidecar
  - target:
      kind: Deployment
      name: myapp
    patch: |-
      - op: replace
        path: /spec/template/spec/containers/1/env/3/valueFrom/secretKeyRef/name
        value: myapp-infisical-secret
      - op: replace
        path: /spec/template/spec/containers/1/env/4/valueFrom/secretKeyRef/name
        value: myapp-infisical-secret
```

**Note**: Les index (`env/3`, `env/4`) correspondent aux positions de `AWS_ACCESS_KEY_ID` et `AWS_SECRET_ACCESS_KEY`. Vérifier avec `kubectl get deployment myapp -o yaml` après application du component.

## Override du mountPath (défaut: /data)

⚠️ **Le component monte par défaut le volume `data` sur `/data`.**

Si votre app utilise un autre path (e.g., Mealie: `/app/data`, Home Assistant: `/config`), ajoutez un patch:

```yaml
# kustomization.yaml
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

**Note**: Les index (`initContainers/0`, `containers/1`) peuvent varier selon votre deployment. Toujours vérifier avec `kubectl get deployment myapp -o yaml` après application.

### Pourquoi ces valeurs sont hardcodées?

Pure kustomize (sans webhook) ne peut pas lire dynamiquement des annotations pour construire des valeurs dans les patches. Le component fournit des **defaults sensibles** que chaque app peut override via patches.

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
