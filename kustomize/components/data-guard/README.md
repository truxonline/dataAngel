# data-guard Kustomize Component (v0.3.0+)

Ce component injecte automatiquement le container **dataangel** (native sidecar init container) dans vos Deployments.

**Architecture**: 1 container unifié avec `restartPolicy: Always`
- Phase 1 (RESTORE): Bloque le démarrage du pod, restore depuis S3
- Phase 2 (BACKUP): Acquiert lock S3, tourne en continu comme sidecar (litestream + rclone)

**Requires**: Kubernetes 1.29+ (native sidecar support)

**New in v0.3.0**: Distributed locking pour RollingUpdate safety (prévient split brain)

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
    data-guard.io/bucket: "my-backup-bucket"              # REQUIS
    data-guard.io/deployment-name: "myapp"                # REQUIS (v0.3.0+, pour distributed lock)
    data-guard.io/sqlite-paths: "/data/app.db"            # Optionnel (si SQLite)
    data-guard.io/fs-paths: "/config"                     # Optionnel (si filesystem)
    data-guard.io/s3-endpoint: "http://minio:9000"        # Optionnel (défaut: AWS S3)
    data-guard.io/aws-region: "us-east-1"                 # Optionnel (auto: us-east-1 si s3-endpoint set)
    data-guard.io/lock-ttl: "60s"                         # Optionnel (défaut: 60s)
    data-guard.io/rclone-interval: "300s"                 # Optionnel (défaut: 60s)
    data-guard.io/metrics-enabled: "true"                 # Optionnel (défaut: true)
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

**Notes importantes:**
- Au moins **un** de `sqlite-paths` ou `fs-paths` doit être défini
- `deployment-name` est **requis** (v0.3.0+) pour le distributed lock RollingUpdate
  - Doit être **unique** par deployment dans le bucket S3
  - Utilisé comme clé de lock: `.locks/{deployment-name}`
- `aws-region` est **auto-defaultée** à `us-east-1` si `s3-endpoint` est set (MinIO, Wasabi, etc.)
  - Utilisateurs **MinIO** : ne pas set cette annotation (default OK)
  - Utilisateurs **AWS S3 réel** : set à votre région (e.g., `eu-west-1`, `us-west-2`)
  - Pourquoi ? AWS SDK v2 nécessite une région même pour S3-compatible non-AWS

## SecurityContext critique (Permissions fichiers)

⚠️ **IMPORTANT** : Le container dataangel **doit tourner avec le même UID/GID que votre application**.

### Pourquoi ?

Les fichiers (SQLite DB, configs) sont partagés via un volume entre :
- dataangel container Phase 1 (restore)
- dataangel container Phase 2 (backup continu)
- Votre app (lecture/écriture)

**Si les UIDs diffèrent** → permission denied, backup/restore échoue.

### Solution

Configurez `securityContext` au niveau **Pod** (s'applique à tous les containers) :

```yaml
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      securityContext:
        runAsUser: 1000    # UID de votre app
        runAsGroup: 1000   # GID de votre app
        fsGroup: 1000      # Propriétaire des volumes
      
      containers:
      - name: myapp
        # Votre app tourne déjà en uid=1000
```

**Trouver l'UID de votre app** :
```bash
# Exec dans votre pod existant (sans data-guard)
kubectl exec -it <pod> -- id
# Exemple output: uid=1000(user) gid=1000(user) groups=1000(user)
```

### Pattern recommandé

```yaml
spec:
  template:
    spec:
      # Pod-level securityContext (s'applique à init, sidecar, et app)
      securityContext:
        runAsUser: <UID_DE_VOTRE_APP>
        runAsGroup: <GID_DE_VOTRE_APP>
        fsGroup: <GID_DE_VOTRE_APP>
        runAsNonRoot: true
```

**Valeurs courantes** :
- Mealie : `uid=911 gid=911`
- Home Assistant : `uid=0 gid=0` (privilégié, nécessite `runAsNonRoot: false`)
- Vaultwarden : `uid=1000 gid=1000`
- Nextcloud : `uid=33 gid=33` (www-data)

⚠️ **Ne PAS hardcoder ces valeurs** — chaque app est différente. Toujours vérifier avec `kubectl exec ... -- id`.

## Secret AWS requis

⚠️ **Le component utilise par défaut un secret nommé `data-guard-credentials`.**

Créez-le dans le namespace de votre app:

```bash
kubectl create secret generic data-guard-credentials \
  --from-literal=access-key=YOUR_ACCESS_KEY \
  --from-literal=secret-key=YOUR_SECRET_KEY
```

### Override du nom du secret

Si votre app utilise un secret différent (e.g., secret Infisical par app), utilisez un **strategic merge patch**:

```yaml
# kustomization.yaml
patches:
  - target:
      kind: Deployment
      name: myapp
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: myapp
      spec:
        template:
          spec:
            initContainers:
              - name: dataangel
                env:
                  - name: AWS_ACCESS_KEY_ID
                    valueFrom:
                      secretKeyRef:
                        name: myapp-infisical-secret
                  - name: AWS_SECRET_ACCESS_KEY
                    valueFrom:
                      secretKeyRef:
                        name: myapp-infisical-secret
```

**Avantages du strategic merge:**
- Merge par **nom** (`name: dataangel`, `name: AWS_ACCESS_KEY_ID`), pas par index
- **Stable**: fonctionne même si l'ordre des env vars change dans le component
- Plus lisible et maintenable

## Override du mountPath (défaut: /data)

⚠️ **Le component monte par défaut le volume `data` sur `/data`.**

Si votre app utilise un autre path (e.g., Mealie: `/app/data`, Home Assistant: `/config`), utilisez un **strategic merge patch**:

```yaml
# kustomization.yaml
patches:
  - target:
      kind: Deployment
      name: myapp
    patch: |-
      apiVersion: apps/v1
      kind: Deployment
      metadata:
        name: myapp
      spec:
        template:
          spec:
            initContainers:
              - name: dataangel
                volumeMounts:
                  - name: data
                    mountPath: /app/data
```

**Avantages du strategic merge:**
- Merge par **nom** (`name: dataangel`, `name: data`), pas par index
- **Stable**: fonctionne même si l'ordre des containers/volumeMounts change
- Un seul patch affecte automatiquement init + sidecar (même container avec `restartPolicy: Always`)

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

Le sidecar peut exposer des métriques sur le port `9090`. Cette fonctionnalité est **optionnelle** et contrôlée par annotation.

### Activer/Désactiver les métriques

**Production** (avec Prometheus):
```yaml
annotations:
  data-guard.io/metrics-enabled: "true"  # Sidecar démarre le metrics server
```

**Dev/CI** (économie resources):
```yaml
annotations:
  data-guard.io/metrics-enabled: "false"  # Sidecar skip le metrics server
```

**Note**: Si l'annotation est absente, le comportement par défaut est `"true"` (backward compatibility).

### Découverte automatique par Prometheus

Pour que Prometheus découvre automatiquement les métriques, utilisez le component **data-guard-monitoring** (opt-in):

```yaml
# kustomization.yaml
components:
  - ../../components/data-guard            # Base component
  - ../../components/data-guard-monitoring # PodMonitor pour Prometheus discovery
```

Ce component ajoute un **PodMonitor** (Prometheus Operator) qui scrape automatiquement les pods avec `data-guard.io/metrics-enabled: "true"`.

**Voir**: [data-guard-monitoring component](../data-guard-monitoring/README.md)

### Debug manuel des métriques

Même sans Prometheus, vous pouvez accéder aux métriques manuellement:

```bash
kubectl port-forward <pod> 9090:9090
curl http://localhost:9090/metrics
```

**Note**: Le port 9090 est toujours défini dans Kubernetes (containerPort), mais le metrics server ne démarre que si `metrics-enabled: "true"`.
