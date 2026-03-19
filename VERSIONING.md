# Versioning Strategy

## Overview

dataAngel uses **Semantic Versioning (semver)** with automatic Docker tagging via GitHub Actions.

## Tag Pattern

### Git Tags → Docker Tags

| Git Tag | Docker Tags Created | Usage |
|---------|---------------------|-------|
| `v0.1.0` | `0.1.0`, `0.1`, `0`, `latest` | Stable release |
| `v0.2.0` | `0.2.0`, `0.2`, `0`, `latest` | Minor update |
| `v1.0.0` | `1.0.0`, `1.0`, `1`, `latest` | Major release |
| (main branch) | `dev`, `sha-abc1234` | Development builds |

### Version Concordance

**GitHub → DockerHub concordance is automatic**. Creating a Git tag triggers GitHub Actions to build and push Docker images with matching version tags.

**Workflow**: `git tag v0.1.0 && git push origin v0.1.0` → GitHub Actions → DockerHub tags created

## Renovate Integration

To track stable versions only (ignore dev builds):

```yaml
# In your Kustomize deployment
images:
  - name: dataangel-init
    newName: charchess/dataangel
    newTag: 0.1.0  # ← Renovate updates this
  - name: dataangel-sidecar
    newName: charchess/dataangel
    newTag: 0.1.0  # ← Renovate updates this
```

### Renovate Configuration

Add to your existing Renovate config (`.github/renovate.json` or `renovate.json`):

```json
{
  "packageRules": [
    {
      "matchDatasources": ["docker"],
      "matchPackageNames": ["charchess/dataangel"],
      "versioning": "semver",
      "allowedVersions": "!/^(dev|sha-)/"
    }
  ]
}
```

This configuration:
- ✅ Tracks semver tags (`0.1.0`, `0.2.0`, `1.0.0`)
- ✅ Follows minor/patch updates automatically
- ❌ Ignores `dev` and `sha-*` tags

## Release Process

### Creating a New Release

1. **Update version files** (if any)
   ```bash
   # No version files currently - all in Git tags
   ```

2. **Create annotated tag**
   ```bash
   git tag -a v0.2.0 -m "Release v0.2.0 - <description>"
   ```

3. **Push tag**
   ```bash
   git push origin v0.2.0
   ```

4. **GitHub Actions builds automatically**
   - Multi-platform build (amd64, arm64)
   - Pushes to DockerHub: `charchess/dataangel:0.2.0`, `0.2`, `0`, `latest`
   - Build logs: https://github.com/truxonline/dataAngel/actions

### Development Workflow

**Main branch** pushes create dev builds automatically:
```bash
git push origin main
# → charchess/dataangel:dev
# → charchess/dataangel:sha-abc1234
```

**Use dev builds for testing**:
```yaml
# Testing deployment
spec:
  initContainers:
  - name: data-guard-init
    image: charchess/dataangel:dev
```

**Use stable versions for production**:
```yaml
# Production deployment
spec:
  initContainers:
  - name: data-guard-init
    image: charchess/dataangel:0.1.0
```

## Version Pinning Recommendations

| Environment | Image Tag | Renovate Behavior |
|-------------|-----------|-------------------|
| Production | `0.1.0` (exact) | Manual approval for updates |
| Staging | `0.1` (minor pin) | Auto-update patch releases |
| Dev/CI | `dev` | Always latest main branch |

## GitHub Actions Workflow

The versioning logic is in `.github/workflows/docker.yml`:

```yaml
tags: |
  # Semver tags on Git tags
  type=semver,pattern={{version}}         # v0.1.0 → 0.1.0
  type=semver,pattern={{major}}.{{minor}} # v0.1.0 → 0.1
  type=semver,pattern={{major}}           # v0.1.0 → 0
  type=raw,value=latest,enable=${{ startsWith(github.ref, 'refs/tags/v') }}
  
  # Dev builds on main
  type=raw,value=dev,enable={{is_default_branch}}
  type=sha,prefix=sha-,format=short,enable={{is_default_branch}}
```

## Version History

| Version | Date | Description |
|---------|------|-------------|
| v0.2.0 | 2026-03-19 | Native sidecar init container (K8s 1.29+), phase metrics, readiness probe |
| v0.1.0 | 2026-03-19 | Initial stable release (5 issues fixed) |

## FAQ

**Q: Why two workflows (`docker.yml` and `build-and-push.yml`)?**  
A: `docker.yml` builds the unified image for DockerHub. `build-and-push.yml` builds separate init/cli images for GHCR (legacy/testing).

**Q: Can I use `latest` in production?**  
A: **No.** Always pin to exact versions (`0.1.0`) or minor versions (`0.1`) for production. `latest` tracks the most recent stable release and can break your deployment on automatic pulls.

**Q: How do I test a PR before merging?**  
A: GitHub Actions builds images on PRs but doesn't push them. Use local Docker build for PR testing:
```bash
docker build -f docker/Dockerfile -t dataangel:pr-test .
```

**Q: How do I rollback a bad release?**  
A: Change your deployment to pin the previous stable version:
```yaml
# Rollback from 0.2.0 to 0.1.0
image: charchess/dataangel:0.1.0
```

Then apply: `kubectl apply -k kustomize/examples/myapp/`
