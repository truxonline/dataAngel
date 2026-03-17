# Déploiement DataGuard dans Kubernetes

## 📦 Images Docker disponibles

Les images sont automatiquement buildées et pushées via GitHub Actions :

- **Init Container**: `ghcr.io/votre-org/dataGuard/data-guard-init`
- **CLI**: `ghcr.io/votre-org/dataGuard/data-guard-cli`

### Utilisation des images

```yaml
# Init Container
image: ghcr.io/votre-org/dataGuard/data-guard-init:latest

# CLI
image: ghcr.io/votre-org/dataGuard/data-guard-cli:latest
```

## 🔧 Configuration Kubernetes

### 1. Secret AWS (nécessaire pour accéder à S3)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: aws-credentials
  namespace: default
type: Opaque
data:
  # Encodé en base64
  access-key: <votre-access-key-base64>
  secret-key: <votre-secret-key-base64>
```

Pour encoder en base64 :
```bash
echo -n "votre-access-key" | base64
echo -n "votre-secret-key" | base64
```

### 2. ConfigMap DataGuard

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: data-guard-config
  namespace: default
data:
  bucket: "mon-bucket-s3"
  region: "eu-west-1"
  checksum: "8932f95bf17cdcc8ed5602e28ff09ccc8967d7ff5ef2f607c4183efae7b2276a"
```

### 3. Déployment avec Init Container DataGuard

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-application
  namespace: default
  annotations:
    data-guard.io/enabled: "true"
    data-guard.io/bucket: "mon-bucket-s3"
    data-guard.io/sqlite-paths: "/data/app.db"
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
      # Init Container DataGuard
      initContainers:
      - name: data-guard-init
        image: ghcr.io/votre-org/dataGuard/data-guard-init:latest
        securityContext:
          runAsUser: 1000
          runAsGroup: 1000
          allowPrivilegeEscalation: false
        env:
        # Configuration S3
        - name: DATA_GUARD_BUCKET
          valueFrom:
            configMapKeyRef:
              name: data-guard-config
              key: bucket
        - name: DATA_GUARD_PATH
          value: "backups/latest.db"
        
        # Configuration locale
        - name: DATA_GUARD_LOCAL_PATH
          value: "/data/app.db"
        
        # Checksum attendu (optionnel mais recommandé)
        - name: DATA_GUARD_CHECKSUM
          valueFrom:
            configMapKeyRef:
              name: data-guard-config
              key: checksum
        
        # Credentials AWS
        - name: AWS_ACCESS_KEY_ID
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: access-key
        - name: AWS_SECRET_ACCESS_KEY
          valueFrom:
            secretKeyRef:
              name: aws-credentials
              key: secret-key
        - name: AWS_REGION
          valueFrom:
            configMapKeyRef:
              name: data-guard-config
              key: region
        
        volumeMounts:
        - name: data-volume
          mountPath: /data
        
        # Pas d'args nécessaires, la config vient des env vars
        command: ["/init-container"]
      
      # Conteneur principal de l'application
      containers:
      - name: my-app
        image: votre-app:latest
        securityContext:
          runAsUser: 1000
          runAsGroup: 1000
        volumeMounts:
        - name: data-volume
          mountPath: /data
        - name: config-volume
          mountPath: /config
        env:
        - name: DATABASE_URL
          value: "/data/app.db"
      
      volumes:
      - name: data-volume
        emptyDir: {}
      - name: config-volume
        configMap:
          name: app-config
```

## 🚀 Déploiement étape par étape

### 1. Préparer les secrets et configmaps

```bash
# Créer le secret AWS
kubectl create secret generic aws-credentials \
  --from-literal=access-key=VOTRE_ACCESS_KEY \
  --from-literal=secret-key=VOTRE_SECRET_KEY

# Créer la ConfigMap DataGuard
kubectl create configmap data-guard-config \
  --from-literal=bucket=mon-bucket-s3 \
  --from-literal=region=eu-west-1 \
  --from-literal=checksum=8932f95bf17cdcc8ed5602e28ff09ccc8967d7ff5ef2f607c4183efae7b2276a
```

### 2. Déployer l'application

```bash
kubectl apply -f deployment.yaml
```

### 3. Vérifier le déploiement

```bash
# Voir les pods
kubectl get pods

# Voir les logs de l'init container
kubectl logs your-pod-name -c data-guard-init

# Voir les logs du conteneur principal
kubectl logs your-pod-name -c my-app
```

### 4. Vérifier les logs attendus

**Succès (restore effectué) :**
```
Starting restore pipeline...
Restore needed: local data is outdated or missing
Restoring data from S3...
Downloaded myapp/backup/data.db to /data/app.db
Restore completed successfully
```

**Succès (skip) :**
```
Starting restore pipeline...
Skipping restore: local data is up to date
```

**Erreur :**
```
Starting restore pipeline...
Error getting local state: ...
```

## 🔄 CI/CD avec GitHub Actions

### Configuration GitHub

1. **Activer GitHub Packages** :
   - Allez dans Settings > Packages
   - Assurez-vous que GitHub Packages est enabled

2. **Permissions** :
   - Le workflow utilise `GITHUB_TOKEN` automatique
   - Pas besoin de tokens supplémentaires

### Workflow automatique

À chaque push sur `main` ou `master` :
1. Les tests sont exécutés
2. Les images Docker sont buildées
3. Les images sont pushées sur GitHub Container Registry
4. Les tags `latest` et `sha` sont créés

### Utilisation des images buildées

```yaml
# Init container
image: ghcr.io/votre-org/dataGuard/data-guard-init:latest

# Ou avec un SHA spécifique
image: ghcr.io/votre-org/dataGuard/data-guard-init:main-abc123
```

## 🔍 Monitoring et débogage

### Vérifier l'état des backups

```bash
# Utiliser le CLI DataGuard
kubectl run data-guard-cli --image=ghcr.io/votre-org/dataGuard/data-guard-cli:latest --rm -i --restart=Never -- verify --bucket mon-bucket-s3
```

### Métriques et logs

- **Init container logs** : `kubectl logs your-pod -c data-guard-init`
- **Exit codes** :
  - `0` : Succès (restore fait ou skip)
  - `1` : Erreur de restore
  - `2` : Erreur de configuration

## ⚠️ Sécurité

- Les images tournent en tant qu'utilisateur non-root (UID 1000)
- `allowPrivilegeEscalation: false` pour plus de sécurité
- Les credentials AWS sont stockés dans des secrets Kubernetes
- Les checksums sont vérifiés pour éviter la corruption de données

## 📝 Exemple complet

Voir `examples/deployment-complete.yaml` pour un exemple prêt à l'emploi.

## 🔧 Maintenance

### Mettre à jour les images

```bash
# Rebuild manuel si nécessaire
docker build -f docker/init-container.Dockerfile -t data-guard-init:latest .
docker push votre-registry/data-guard-init:latest
```

### Nettoyer les anciennes versions

```bash
# Garder les 5 dernières versions
kubectl get pods -l app=my-application
```

## ✅ Checklist de déploiement

- [ ] Secret AWS créé
- [ ] ConfigMap DataGuard configuré
- [ ] Images Docker disponibles sur le registry
- [ ] Déployment YAML prêt
- [ ] Tests d'intégration passés
- [ ] Logs vérifiés
- [ ] Exit codes corrects

---

**Le projet est maintenant prêt pour la production !** 🚀