package main

import (
	"strings"
	"testing"
)

func TestGenerateK8sSidecarSpec_ValidSpec(t *testing.T) {
	// Arrange
	volumeMountPath := "/data"
	s3Bucket := "my-backup-bucket"
	s3SecretName := "s3-credentials"

	// Act
	spec, err := GenerateK8sSidecarSpec(volumeMountPath, s3Bucket, s3SecretName)

	// Assert
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	if !strings.Contains(spec, "containers:") {
		t.Errorf("Expected spec to contain containers section, got: %s", spec)
	}
	if !strings.Contains(spec, "litestream") {
		t.Errorf("Expected spec to contain litestream container, got: %s", spec)
	}
}

func TestGenerateK8sSidecarSpec_MissingVolume(t *testing.T) {
	// Act
	_, err := GenerateK8sSidecarSpec("", "bucket", "secret")

	// Assert
	if err == nil {
		t.Error("Expected error for missing volume")
	}
	if !strings.Contains(err.Error(), "volume mount path is required") {
		t.Errorf("Expected error about volume mount path, got: %v", err)
	}
}

func TestGenerateK8sSidecarSpec_MissingS3Config(t *testing.T) {
	// Act
	_, err := GenerateK8sSidecarSpec("/data", "", "secret")

	// Assert
	if err == nil {
		t.Error("Expected error for missing S3 config")
	}
	if !strings.Contains(err.Error(), "S3 bucket is required") {
		t.Errorf("Expected error about S3 bucket, got: %v", err)
	}
}

func TestGenerateVolumeMountSpec_ValidMount(t *testing.T) {
	// Arrange
	volumeName := "sqlite-data"
	mountPath := "/data"

	// Act
	mount := GenerateVolumeMountSpec(volumeName, mountPath)

	// Assert
	if !strings.Contains(mount, "name: "+volumeName) {
		t.Errorf("Expected mount to contain volume name, got: %s", mount)
	}
	if !strings.Contains(mount, "mountPath: "+mountPath) {
		t.Errorf("Expected mount to contain mount path, got: %s", mount)
	}
}

func TestGenerateEnvFromSecret_ValidSecret(t *testing.T) {
	// Arrange
	secretName := "s3-credentials"

	// Act
	env := GenerateEnvFromSecret(secretName)

	// Assert
	if !strings.Contains(env, "name: "+secretName) {
		t.Errorf("Expected env to contain secret name, got: %s", env)
	}
}
