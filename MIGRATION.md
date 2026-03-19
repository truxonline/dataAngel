# Migration Guide

## v0.4.0 - Naming Consistency (Breaking Changes)

**Release:** v0.4.0  
**Issue:** #15

### Summary

The project API has been renamed from `data-guard.io` to `dataangel.io` for consistency with the project name **dataAngel**.

### Breaking Changes

| Old Name | New Name |
|----------|----------|
| `data-guard.io/*` annotations | `dataangel.io/*` annotations |
| `data-guard-credentials` secret | `dataangel-credentials` secret |
| `components/data-guard` | `components/dataangel` |
| `components/data-guard-monitoring` | `components/dataangel-monitoring` |
| `data-guard.io/enabled` label | `dataangel.io/enabled` label |

### Migration Steps

#### 1. Update Kustomize Component Reference

```yaml
# Before (v0.3.x)
components:
  - ../../components/data-guard

# After (v0.4.0+)
components:
  - ../../components/dataangel
```

#### 2. Update Deployment Annotations

```yaml
# Before (v0.3.x)
metadata:
  annotations:
    data-guard.io/bucket: "my-bucket"
    data-guard.io/deployment-name: "myapp"
    data-guard.io/sqlite-paths: "/data/app.db"
    data-guard.io/s3-endpoint: "http://minio:9000"
    data-guard.io/aws-region: "us-east-1"
    data-guard.io/lock-ttl: "60s"
    data-guard.io/rclone-interval: "300s"
    data-guard.io/metrics-enabled: "true"

# After (v0.4.0+)
metadata:
  annotations:
    dataangel.io/bucket: "my-bucket"
    dataangel.io/deployment-name: "myapp"
    dataangel.io/sqlite-paths: "/data/app.db"
    dataangel.io/s3-endpoint: "http://minio:9000"
    dataangel.io/aws-region: "us-east-1"
    dataangel.io/lock-ttl: "60s"
    dataangel.io/rclone-interval: "300s"
    dataangel.io/metrics-enabled: "true"
```

#### 3. Rename Secret

**Option A:** Rename existing secret (preserves credentials)
```bash
kubectl get secret data-guard-credentials -n <namespace> -o yaml | \
  sed 's/name: data-guard-credentials/name: dataangel-credentials/' | \
  kubectl apply -f -

# Delete old secret after verification
kubectl delete secret data-guard-credentials -n <namespace>
```

**Option B:** Use strategic merge patch to override secret name

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
                        name: my-custom-secret  # Your existing secret
                  - name: AWS_SECRET_ACCESS_KEY
                    valueFrom:
                      secretKeyRef:
                        name: my-custom-secret
```

#### 4. Update Image Tag

```yaml
# Use v0.4.0+ for the new naming
image: charchess/dataangel:0.4.0
```

### Rollback (if needed)

If you need to rollback to v0.3.x:

1. Revert component reference: `components/dataangel` → `components/data-guard`
2. Revert annotations: `dataangel.io/*` → `data-guard.io/*`
3. Revert secret name: `dataangel-credentials` → `data-guard-credentials`
4. Pin image tag: `charchess/dataangel:0.3.1`

### Why This Change?

**Before:** Users deploying **dataAngel** saw `data-guard.io` everywhere (annotations, secrets, components) with no obvious connection to the project name.

**After:** Consistent `dataangel.io` branding aligns with the project name.

### Support

- v0.3.x: Uses `data-guard.io` (legacy, still functional)
- v0.4.0+: Uses `dataangel.io` (new standard)

No backward compatibility layer is provided. You must update your manifests when upgrading to v0.4.0+.
