# Mealie avec data-guard

Exemple d'intégration du component data-guard pour Mealie.

## Spécificités Mealie

- **mountPath**: `/app/data` (pas `/data`)
- **SQLite DB**: `/app/data/mealie.db`
- **Filesystem**: `/app/data/recipes`, `/app/data/user-files`
- **Deployment name**: `mealie` (pour distributed lock)
- **Secret**: `mealie-infisical-secret` (géré par Infisical)
- **UID/GID**: `911:911` (user linuxserver.io standard)

### Patches (Strategic Merge)

Le `kustomization.yaml` utilise **strategic merge patches** pour override:
- `volumeMount.mountPath`: `/app/data` (au lieu du défaut `/data`)
- `secret name`: `mealie-infisical-secret` (au lieu de `data-guard-credentials`)

Pattern stable par **nom** (`name: dataangel`, `name: AWS_ACCESS_KEY_ID`), pas par index.

### SecurityContext

⚠️ **CRITIQUE** : L'image Mealie tourne en `uid=911 gid=911`. Les containers data-guard (init + sidecar) doivent utiliser le **même UID** pour accéder aux fichiers partagés.

Le deployment inclut :
```yaml
securityContext:
  runAsUser: 911
  runAsGroup: 911
  fsGroup: 911
  runAsNonRoot: true
```

**Pourquoi c'est critique** : Sans ça, `permission denied` sur la DB/config.

## Déploiement

```bash
# Créer le namespace
kubectl create namespace mealie

# Créer le PVC (ajuster selon votre storage class)
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: mealie-data
  namespace: mealie
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
EOF

# Vérifier que le secret Infisical existe
kubectl get secret mealie-infisical-secret -n mealie

# Si pas encore créé, créer un secret temporaire pour test
kubectl create secret generic mealie-infisical-secret \
  --from-literal=access-key=YOUR_MINIO_KEY \
  --from-literal=secret-key=YOUR_MINIO_SECRET \
  -n mealie

# Déployer avec kustomize
kubectl apply -k .

# Vérifier les logs
kubectl logs -n mealie -l app=mealie -c data-guard-init
kubectl logs -n mealie -l app=mealie -c data-guard-sidecar -f
```

## Vérification

```bash
# Vérifier que la DB est restaurée
kubectl exec -n mealie deploy/mealie -c mealie -- ls -lh /app/data/mealie.db

# Vérifier les métriques (si metrics-enabled: "true")
kubectl port-forward -n mealie deploy/mealie 9090:9090
curl http://localhost:9090/metrics | grep dataguard
```

## Monitoring (optionnel)

Pour activer la découverte automatique par Prometheus, ajoutez le component monitoring:

```yaml
# kustomization.yaml
components:
  - ../../components/data-guard
  - ../../components/data-guard-monitoring  # PodMonitor pour Prometheus
```

**Prérequis**: Prometheus Operator installé (CRD `monitoring.coreos.com/v1`).

Voir [data-guard-monitoring](../../components/data-guard-monitoring/README.md) pour plus de détails.

## Backup initial

Si c'est la première installation, pousser une backup initiale:

```bash
# Copier la DB locale
kubectl cp mealie.db mealie/mealie-xxx:/app/data/mealie.db

# Attendre que le sidecar backup vers MinIO (~1min)
kubectl logs -n mealie -l app=mealie -c data-guard-sidecar -f
```
