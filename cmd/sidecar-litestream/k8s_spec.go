package main

import (
	"fmt"
)

// GenerateK8sSidecarSpec generates a Kubernetes sidecar container spec
func GenerateK8sSidecarSpec(volumeMountPath, s3Bucket, s3SecretName string) (string, error) {
	if volumeMountPath == "" {
		return "", fmt.Errorf("volume mount path is required")
	}
	if s3Bucket == "" {
		return "", fmt.Errorf("S3 bucket is required")
	}

	spec := fmt.Sprintf(`containers:
  - name: litestream
    image: litestream/litestream:latest
    volumeMounts:
      - name: sqlite-data
        mountPath: %s
        readOnly: true
    envFrom:
      - secretRef:
          name: %s
    command: ["/bin/sh"]
    args: ["-c", "litestream replicate"]
`, volumeMountPath, s3SecretName)

	return spec, nil
}

// GenerateVolumeMountSpec generates a volume mount specification
func GenerateVolumeMountSpec(volumeName, mountPath string) string {
	return fmt.Sprintf(`volumeMounts:
  - name: %s
    mountPath: %s
    readOnly: true
`, volumeName, mountPath)
}

// GenerateEnvFromSecret generates environment variables from a secret
func GenerateEnvFromSecret(secretName string) string {
	return fmt.Sprintf(`envFrom:
  - secretRef:
      name: %s
`, secretName)
}
