package main

import (
	"fmt"
	"strings"
)

// GenerateDockerfile creates a Dockerfile for the Litestream sidecar
func GenerateDockerfile(image string, configs ...string) (string, error) {
	if image == "" {
		return "", fmt.Errorf("image is required")
	}

	var lines []string
	lines = append(lines, fmt.Sprintf("FROM %s", image))
	lines = append(lines, "")

	// Copy configuration files if provided
	for _, config := range configs {
		lines = append(lines, fmt.Sprintf("COPY %s /etc/litestream.yml", config))
	}

	lines = append(lines, "")
	lines = append(lines, "# Health check")
	lines = append(lines, "HEALTHCHECK CMD litestream status || exit 1")

	return strings.Join(lines, "\n"), nil
}

// GenerateLitestreamConfig creates a Litestream configuration
func GenerateLitestreamConfig(dbPath, s3Bucket, s3Path string) (string, error) {
	if dbPath == "" {
		return "", fmt.Errorf("database path is required")
	}
	if s3Bucket == "" {
		return "", fmt.Errorf("S3 bucket is required")
	}
	if s3Path == "" {
		return "", fmt.Errorf("S3 path is required")
	}

	if !strings.HasPrefix(s3Path, "/") {
		s3Path = "/" + s3Path
	}

	config := fmt.Sprintf(`db: %s
replicas:
  - type: s3
    bucket: %s
    path: %s
    s3_uri: s3://%s%s
`, dbPath, s3Bucket, s3Path, s3Bucket, s3Path)

	return config, nil
}

// GenerateHealthCheck creates a health check script
func GenerateHealthCheck() string {
	return `#!/bin/sh
litestream status
`
}
