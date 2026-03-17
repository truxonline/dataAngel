# Epic 1 - Initial Setup & Data Discovery

## Statut Actuel
- **Epic 1**: In Progress (1/4 stories completed)
- **Story 1.1**: ✅ Done (Configurer DataGuard via annotations K8s)

## Stories
1. **1.1: Configurer DataGuard via annotations K8s** ✅
   - Implémentation du parser d'annotations (`internal/k8s/annotations.go`)
   - Tests TDD complets (6 tests passants)
   - Intégration dans sidecar-litestream
   - Kustomize component pour injection conditionnelle

2. **1.2: Init container detect healthy data** ⏳
   - À implémenter

3. **1.3: Restore conditionnel ou skip** ⏳
   - À implémenter

4. **1.4: CLI verify backup state** ⏳
   - Déjà implémenté dans Epic 6 (à vérifier)

## Annotations Supportées
- `data-guard.io/enabled` (requis) - Activer DataGuard
- `data-guard.io/bucket` (requis) - Bucket S3
- `data-guard.io/sqlite-paths` (optionnel) - Chemins DB
- `data-guard.io/fs-paths` (optionnel) - Chemins répertoires
- `data-guard.io/s3-endpoint` (optionnel) - Override endpoint S3
- `data-guard.io/full-logs` (optionnel) - Logging verbose

## Exemple d'Utilisation
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: my-app
  annotations:
    data-guard.io/enabled: "true"
    data-guard.io/bucket: "my-bucket"
    data-guard.io/sqlite-paths: "/data/app.db"
    data-guard.io/fs-paths: "/config/settings.yaml"
spec:
  template:
    spec:
      containers:
      - name: my-app
        image: my-app:latest
```

## Prochaine Étape
Implémenter la Story 1.2 (Init container detect healthy data).
