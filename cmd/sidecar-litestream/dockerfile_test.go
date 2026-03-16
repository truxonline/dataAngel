package main

import (
	"strings"
	"testing"
)

func TestGenerateDockerfile_LitestreamBase(t *testing.T) {

	// Arrange
	image := "litestream/litestream:latest"

	// Act
	dockerfile, err := GenerateDockerfile(image)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(dockerfile, "FROM "+image) {
		t.Errorf("Expected Dockerfile to contain 'FROM %s', got: %s", image, dockerfile)
	}
}

func TestGenerateDockerfile_WithLitestreamConfig(t *testing.T) {

	// Arrange
	image := "litestream/litestream:latest"
	configPath := "/etc/litestream.yml"

	// Act
	dockerfile, err := GenerateDockerfile(image, configPath)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(dockerfile, "COPY "+configPath) {
		t.Errorf("Expected Dockerfile to copy config file, got: %s", dockerfile)
	}
}

func TestGenerateLitestreamConfig_ValidS3(t *testing.T) {
	// Arrange
	dbPath := "/data/app.db"
	s3Bucket := "my-backup-bucket"
	s3Path := "/backups/app"

	// Act
	config, err := GenerateLitestreamConfig(dbPath, s3Bucket, s3Path)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(config, "db: "+dbPath) {
		t.Errorf("Expected config to contain db path, got: %s", config)
	}
	if !strings.Contains(config, "bucket: "+s3Bucket) {
		t.Errorf("Expected config to contain bucket, got: %s", config)
	}
	if !strings.Contains(config, "path: "+s3Path) {
		t.Errorf("Expected config to contain path, got: %s", config)
	}
}

func TestGenerateLitestreamConfig_MissingDB(t *testing.T) {
	// Act
	_, err := GenerateLitestreamConfig("", "my-backup-bucket", "/backups/app")

	// Assert
	if err == nil {
		t.Error("Expected error for missing database path")
	}
	if !strings.Contains(err.Error(), "database path is required") {
		t.Errorf("Expected error message about database path, got: %v", err)
	}
}

func TestGenerateHealthCheck_Script(t *testing.T) {

	// Act
	healthCheck := GenerateHealthCheck()

	// Assert
	if !strings.Contains(healthCheck, "litestream") {
		t.Errorf("Expected health check to reference litestream, got: %s", healthCheck)
	}
}
