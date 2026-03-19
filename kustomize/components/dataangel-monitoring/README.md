# dataangel-monitoring Component

Component optionnel pour activer la découverte automatique des métriques data-guard par Prometheus Operator.

## Prérequis

- Prometheus Operator installé (CRDs `monitoring.coreos.com/v1`)
- Component `data-guard` déjà appliqué
- Annotation `dataangel.io/metrics-enabled: "true"` sur les pods

## Ce que fait ce component

Ajoute un **PodMonitor** qui :
- Découvre automatiquement les pods data-guard
- Scrape les métriques sur le port `metrics` (9090)
- Interval de scrape: 30s
- Path: `/metrics`

## Usage

### Environnement AVEC Prometheus Operator (production)

```yaml
# kustomization.yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Kustomization

components:
  - ../../components/data-guard            # Base component
  - ../../components/dataangel-monitoring # Monitoring opt-in

resources:
  - deployment.yaml
```

**Deployment**:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  template:
    metadata:
      annotations:
        dataangel.io/bucket: "my-bucket"
        dataangel.io/sqlite-paths: "/data/app.db"
        dataangel.io/metrics-enabled: "true"  # ← Requis pour monitoring
```

### Environnement SANS Prometheus Operator (dev, CI)

```yaml
# kustomization.yaml
components:
  - ../../components/data-guard  # Monitoring component OMIS

resources:
  - deployment.yaml
```

**Deployment**:
```yaml
metadata:
  annotations:
    dataangel.io/metrics-enabled: "false"  # ← Désactive metrics server
```

## Métriques exposées

Le sidecar data-guard expose les métriques suivantes sur `/metrics`:

### Litestream
- `dataguard_litestream_up` (gauge) - Litestream running status
- `dataguard_litestream_replicas` (gauge) - Active Litestream replicas

### Rclone
- `dataguard_rclone_up` (gauge) - Rclone daemon status
- `dataguard_rclone_syncs_total` (counter) - Total rclone syncs
- `dataguard_rclone_syncs_failed_total` (counter) - Failed syncs
- `dataguard_rclone_sync_duration_seconds` (histogram) - Sync duration

### Général
- `dataguard_sidecar_uptime_seconds` (gauge) - Sidecar uptime
- `dataguard_yaml_validations_total` (counter) - YAML validations
- `dataguard_yaml_cache_hits_total` (counter) - YAML cache hits

## Labels Prometheus

Le PodMonitor inclut les labels suivants pour la découverte:

```yaml
labels:
  release: prometheus  # Requis par kube-prometheus-stack
  app.kubernetes.io/name: data-guard
  app.kubernetes.io/component: monitoring
```

**Note**: Si votre Prometheus utilise un autre label selector, modifiez `podmonitor.yaml` en conséquence.

Vérifier le selector:
```bash
kubectl get prometheus -n monitoring -o jsonpath='{.items[0].spec.podMonitorSelector}'
```

## Troubleshooting

### PodMonitor créé mais pas de métriques dans Prometheus

1. Vérifier que le pod a l'annotation `metrics-enabled: "true"`:
   ```bash
   kubectl get pod <pod> -o jsonpath='{.metadata.annotations}'
   ```

2. Vérifier que le sidecar a démarré le metrics server:
   ```bash
   kubectl logs <pod> -c data-guard-sidecar | grep "Starting metrics server"
   ```

3. Tester manuellement le endpoint metrics:
   ```bash
   kubectl port-forward <pod> 9090:9090
   curl http://localhost:9090/metrics
   ```

4. Vérifier que Prometheus découvre le PodMonitor:
   ```bash
   kubectl get podmonitor -A
   kubectl describe podmonitor data-guard-sidecar
   ```

### Erreur: The Kubernetes API could not find monitoring.coreos.com/v1

**Cause**: Prometheus Operator pas installé.

**Solution**: 
- Environnement dev/CI → Retirer ce component
- Environnement prod → Installer kube-prometheus-stack:
  ```bash
  helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
  helm install prometheus prometheus-community/kube-prometheus-stack -n monitoring
  ```

### Métriques exposées mais pas scrapées

Vérifier les labels du PodMonitor matchent le `podMonitorSelector` de Prometheus:

```bash
# Voir le selector Prometheus
kubectl get prometheus -n monitoring -o yaml | grep -A5 podMonitorSelector

# Adapter podmonitor.yaml si nécessaire
```

## Pattern dev vs prod

### Dev/CI (économie resources)
```yaml
annotations:
  dataangel.io/metrics-enabled: "false"

# Pas de component monitoring
```

### Prod (observabilité complète)
```yaml
annotations:
  dataangel.io/metrics-enabled: "true"

components:
  - dataangel-monitoring  # PodMonitor discovery
```

## Voir aussi

- [Prometheus Operator Documentation](https://prometheus-operator.dev/)
- [PodMonitor API Reference](https://prometheus-operator.dev/docs/operator/api/#monitoring.coreos.com/v1.PodMonitor)
- [data-guard base component](../data-guard/README.md)
