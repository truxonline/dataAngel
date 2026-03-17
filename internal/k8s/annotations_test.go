package k8s

import (
	"testing"
)

func TestParseAnnotations_Disabled(t *testing.T) {
	annotations := map[string]string{
		"data-guard.io/enabled": "false",
	}

	config, err := ParseAnnotations(annotations)
	if err != nil {
		t.Errorf("ParseAnnotations ne devrait pas retourner d'erreur: %v", err)
	}

	if config.Enabled {
		t.Errorf("Enabled devrait être false")
	}
}

func TestParseAnnotations_EnabledWithBucket(t *testing.T) {
	annotations := map[string]string{
		"data-guard.io/enabled": "true",
		"data-guard.io/bucket":  "test-bucket",
	}

	config, err := ParseAnnotations(annotations)
	if err != nil {
		t.Errorf("ParseAnnotations ne devrait pas retourner d'erreur: %v", err)
	}

	if !config.Enabled {
		t.Errorf("Enabled devrait être true")
	}

	if config.Bucket != "test-bucket" {
		t.Errorf("Bucket devrait être 'test-bucket', got '%s'", config.Bucket)
	}
}

func TestParseAnnotations_MissingBucket(t *testing.T) {
	annotations := map[string]string{
		"data-guard.io/enabled": "true",
	}

	_, err := ParseAnnotations(annotations)
	if err == nil {
		t.Errorf("ParseAnnotations devrait retourner une erreur quand bucket manquant")
	}
}

func TestParseAnnotations_WithPaths(t *testing.T) {
	annotations := map[string]string{
		"data-guard.io/enabled":      "true",
		"data-guard.io/bucket":       "test-bucket",
		"data-guard.io/sqlite-paths": "/data/app.db,/data/other.db",
		"data-guard.io/fs-paths":     "/data/config.yaml,/data/secrets.yaml",
		"data-guard.io/s3-endpoint":  "https://minio.local",
		"data-guard.io/full-logs":    "true",
	}

	config, err := ParseAnnotations(annotations)
	if err != nil {
		t.Errorf("ParseAnnotations ne devrait pas retourner d'erreur: %v", err)
	}

	if len(config.SqlitePaths) != 2 {
		t.Errorf("SqlitePaths devrait avoir 2 éléments, got %d", len(config.SqlitePaths))
	}

	if len(config.FsPaths) != 2 {
		t.Errorf("FsPaths devrait avoir 2 éléments, got %d", len(config.FsPaths))
	}

	if config.S3Endpoint != "https://minio.local" {
		t.Errorf("S3Endpoint devrait être 'https://minio.local', got '%s'", config.S3Endpoint)
	}

	if !config.FullLogs {
		t.Errorf("FullLogs devrait être true")
	}
}

func TestToEnvVars(t *testing.T) {
	config := &DataGuardConfig{
		Enabled:     true,
		Bucket:      "test-bucket",
		S3Endpoint:  "https://minio.local",
		SqlitePaths: []string{"/data/app.db"},
		FsPaths:     []string{"/data/config.yaml"},
		FullLogs:    true,
	}

	envVars := config.ToEnvVars()

	expectedVars := map[string]bool{
		"DATA_GUARD_ENABLED=true":       false,
		"DATA_GUARD_BUCKET=test-bucket": false,
	}

	for _, envVar := range envVars {
		if _, ok := expectedVars[envVar]; ok {
			expectedVars[envVar] = true
		}
	}

	for expected, found := range expectedVars {
		if !found {
			t.Errorf("Variable d'environnement attendue non trouvée: %s", expected)
		}
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("DATA_GUARD_ENABLED", "true")
	t.Setenv("DATA_GUARD_BUCKET", "env-bucket")

	config, err := LoadFromEnv()
	if err != nil {
		t.Errorf("LoadFromEnv ne devrait pas retourner d'erreur: %v", err)
	}

	if !config.Enabled {
		t.Errorf("Enabled devrait être true")
	}

	if config.Bucket != "env-bucket" {
		t.Errorf("Bucket devrait être 'env-bucket', got '%s'", config.Bucket)
	}
}
