# Home Assistant avec data-guard

Exemple d'intégration du component data-guard pour Home Assistant.

## Spécificités Home Assistant

- **mountPath**: `/config` (pas `/data`)
- **SQLite DB**: `/config/home-assistant_v2.db`
- **Filesystem**: `/config` (tout le répertoire config)
- **Secret**: `home-assistant-infisical-secret` (géré par Infisical)
- **Network**: `hostNetwork: true` (nécessaire pour découverte locale)
- **UID/GID**: `0:0` (root, nécessaire pour hardware access)

### SecurityContext

⚠️ **CRITIQUE** : Home Assistant nécessite `root` pour accéder au hardware (USB, GPIO, etc.). Les containers data-guard (init + sidecar) doivent aussi tourner en `root` pour accéder aux fichiers partagés.

Le deployment inclut :
```yaml
securityContext:
  runAsUser: 0
  runAsGroup: 0
  fsGroup: 0
  runAsNonRoot: false
```

**Pourquoi c'est critique** : Sans ça, `permission denied` sur `/config`.

## Déploiement

```bash
# Créer le namespace
kubectl create namespace home-assistant

# Créer le PVC
kubectl apply -f - <<EOF
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: home-assistant-config
  namespace: home-assistant
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 5Gi
EOF

# Vérifier que le secret Infisical existe
kubectl get secret home-assistant-infisical-secret -n home-assistant

# Si pas encore créé, créer un secret temporaire pour test
kubectl create secret generic home-assistant-infisical-secret \
  --from-literal=access-key=YOUR_MINIO_KEY \
  --from-literal=secret-key=YOUR_MINIO_SECRET \
  -n home-assistant

# Déployer avec kustomize
kubectl apply -k .

# Vérifier les logs
kubectl logs -n home-assistant -l app=home-assistant -c data-guard-init
kubectl logs -n home-assistant -l app=home-assistant -c data-guard-sidecar -f
```

## Vérification

```bash
# Vérifier que la DB est restaurée
kubectl exec -n home-assistant deploy/home-assistant -c home-assistant -- \
  ls -lh /config/home-assistant_v2.db

# Vérifier les fichiers config
kubectl exec -n home-assistant deploy/home-assistant -c home-assistant -- \
  ls -lh /config/*.yaml

# Vérifier les métriques (si metrics-enabled: "true")
kubectl port-forward -n home-assistant deploy/home-assistant 9090:9090
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

## Notes importantes

### Exclusion des fichiers .db dans filesystem backup

Le `DATA_GUARD_FS_PATHS` pointe sur `/config`, qui contient **aussi** la DB SQLite. Mais rclone utilise `--exclude "*.db*"` pour éviter de doubler le backup de la DB (déjà géré par Litestream).

Résultat:
- Litestream backup: `/config/home-assistant_v2.db` → `s3://bucket/home-assistant_v2.db`
- Rclone backup: `/config/*` (sauf *.db*) → `s3://bucket/config/`

### Restore behavior

Au restore (init container):
1. Litestream restore la DB: `home-assistant_v2.db`
2. Rclone restore les autres fichiers: `configuration.yaml`, `automations.yaml`, etc.

### Backup initial

Si première installation, pousser une backup initiale:

```bash
# Copier votre config locale
kubectl cp config/ home-assistant/home-assistant-xxx:/config/

# Attendre que le sidecar backup vers MinIO
kubectl logs -n home-assistant -l app=home-assistant -c data-guard-sidecar -f
```
