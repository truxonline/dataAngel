# data-guard Component - Exemples Production

Exemples d'intégration du component data-guard pour apps courantes.

## Exemples disponibles

### Mealie

[`mealie/`](./mealie/)

- **mountPath**: `/app/data`
- **SQLite**: `/app/data/mealie.db`
- **Filesystem**: `/app/data/recipes`, `/app/data/user-files`
- **Spécificités**: PVC pour données, secret Infisical

### Home Assistant

[`home-assistant/`](./home-assistant/)

- **mountPath**: `/config`
- **SQLite**: `/config/home-assistant_v2.db`
- **Filesystem**: `/config` (tout le répertoire)
- **Spécificités**: `hostNetwork: true`, DB + config dans même volume

## Pattern commun

Tous les exemples suivent ce pattern:

1. **Component générique**: `../../components/data-guard`
2. **Annotations**: Configurent bucket, paths SQLite/FS, endpoint S3, deployment name
3. **Strategic Merge Patches** (stable, merge par nom):
   - Override `mountPath` (chaque app utilise un path différent)
   - Override `secret name` (chaque app a son secret Infisical)

**Pourquoi strategic merge?** Merge par **nom** (`name: dataangel`), pas par index. Stable même si le component change l'ordre des containers/env vars.

## Adapter pour votre app

```bash
# Copier un exemple existant
cp -r mealie/ my-app/
cd my-app/

# Modifier deployment.yaml:
# - Ajuster annotations data-guard.io/* (IMPORTANT: deployment-name unique!)
# - Ajuster securityContext (runAsUser/runAsGroup selon votre app)
# - Ajuster image, ports, env vars de votre app

# Modifier kustomization.yaml:
# - Changer namespace
# - Patch: changer mountPath (si différent de /data)
# - Patch: changer volumeMount name (si différent de "data")
# - Patch: changer secret name (si différent de data-guard-credentials)

# Déployer
kubectl apply -k .
```

**Checklist critique:**
- ✅ `deployment-name` annotation unique par deployment
- ✅ `runAsUser`/`runAsGroup` matchent l'UID de votre app
- ✅ Volume name dans patch correspond au volume dans deployment
- ✅ Au moins un de `sqlite-paths` ou `fs-paths` défini

## Limitations kustomize

⚠️ **Pourquoi ces patches manuels?**

Pure kustomize (sans webhook) ne peut pas lire dynamiquement des annotations pour construire des valeurs. Le component fournit des **defaults sensibles** (`/data`, `data-guard-credentials`) que chaque app override via patches.

## Alternative: Component par app

Si vous avez beaucoup d'apps, créer un component par app évite les patches répétitifs:

```bash
# Structure alternative
kustomize/
  components/
    data-guard/           # Component générique (base)
    data-guard-mealie/    # Component spécifique Mealie
    data-guard-hass/      # Component spécifique Home Assistant
```

Chaque component spécifique:
1. Inclut le component générique
2. Applique les patches spécifiques

Exemple `data-guard-mealie/kustomization.yaml`:

```yaml
apiVersion: kustomize.config.k8s.io/v1alpha1
kind: Component

components:
  - ../data-guard

patches:
  # Patches spécifiques Mealie (mountPath, secret)
```

Puis dans votre app:

```yaml
# mealie/kustomization.yaml
components:
  - ../../components/data-guard-mealie  # Au lieu de data-guard + patches
```

Avantages:
- Pas de patches répétitifs par app
- Configuration centralisée par type d'app
- Plus maintenable

Inconvénients:
- Plus de components à maintenir
- Coupling entre component et apps spécifiques
