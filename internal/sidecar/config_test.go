package sidecar

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromEnv(t *testing.T) {
	t.Run("should load all required environment variables", func(t *testing.T) {
		// ARRANGE
		t.Setenv("DATA_GUARD_BUCKET", "test-bucket")
		t.Setenv("DATA_GUARD_S3_ENDPOINT", "http://minio:9000")
		t.Setenv("DATA_GUARD_SQLITE_PATHS", "/data/db1.sqlite,/data/db2.sqlite")
		t.Setenv("DATA_GUARD_FS_PATHS", "/config,/data/files")
		t.Setenv("DATA_GUARD_YAML_PATHS", "/config/*.yaml,/data/*.yml")

		// ACT
		config, err := LoadFromEnv()

		// ASSERT
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if config.Bucket != "test-bucket" {
			t.Errorf("Expected bucket 'test-bucket', got '%s'", config.Bucket)
		}
		if config.S3Endpoint != "http://minio:9000" {
			t.Errorf("Expected endpoint 'http://minio:9000', got '%s'", config.S3Endpoint)
		}
		if len(config.SqlitePaths) != 2 {
			t.Errorf("Expected 2 sqlite paths, got %d", len(config.SqlitePaths))
		}
		if len(config.FsPaths) != 2 {
			t.Errorf("Expected 2 fs paths, got %d", len(config.FsPaths))
		}
		if len(config.YAMLPaths) != 2 {
			t.Errorf("Expected 2 yaml paths, got %d", len(config.YAMLPaths))
		}
	})

	t.Run("should fail when DATA_GUARD_BUCKET is missing", func(t *testing.T) {
		// ARRANGE
		os.Unsetenv("DATA_GUARD_BUCKET")

		// ACT
		_, err := LoadFromEnv()

		// ASSERT
		if err == nil {
			t.Error("Expected error when DATA_GUARD_BUCKET is missing")
		}
	})

	t.Run("should use default metrics port 9090", func(t *testing.T) {
		// ARRANGE
		t.Setenv("DATA_GUARD_BUCKET", "test-bucket")
		os.Unsetenv("DATA_GUARD_METRICS_PORT")

		// ACT
		config, err := LoadFromEnv()

		// ASSERT
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.MetricsPort != 9090 {
			t.Errorf("Expected default port 9090, got %d", config.MetricsPort)
		}
	})

	t.Run("should use default shutdown timeout 15s", func(t *testing.T) {
		// ARRANGE
		t.Setenv("DATA_GUARD_BUCKET", "test-bucket")
		os.Unsetenv("DATA_GUARD_SHUTDOWN_TIMEOUT")

		// ACT
		config, err := LoadFromEnv()

		// ASSERT
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if config.ShutdownTimeout != 15*time.Second {
			t.Errorf("Expected default 15s, got %v", config.ShutdownTimeout)
		}
	})

	t.Run("should parse CSV sqlite paths correctly", func(t *testing.T) {
		// ARRANGE
		t.Setenv("DATA_GUARD_BUCKET", "test-bucket")
		t.Setenv("DATA_GUARD_SQLITE_PATHS", "/db1.sqlite, /db2.sqlite , /db3.sqlite")

		// ACT
		config, err := LoadFromEnv()

		// ASSERT
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(config.SqlitePaths) != 3 {
			t.Errorf("Expected 3 paths, got %d", len(config.SqlitePaths))
		}
		// Check trimmed values
		if config.SqlitePaths[0] != "/db1.sqlite" {
			t.Errorf("Expected '/db1.sqlite', got '%s'", config.SqlitePaths[0])
		}
	})

	t.Run("should parse CSV yaml paths correctly", func(t *testing.T) {
		// ARRANGE
		t.Setenv("DATA_GUARD_BUCKET", "test-bucket")
		t.Setenv("DATA_GUARD_YAML_PATHS", "*.yaml, /config/*.yml")

		// ACT
		config, err := LoadFromEnv()

		// ASSERT
		if err != nil {
			t.Fatalf("Unexpected error: %v", err)
		}
		if len(config.YAMLPaths) != 2 {
			t.Errorf("Expected 2 patterns, got %d", len(config.YAMLPaths))
		}
	})
}
