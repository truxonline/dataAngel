package sidecar

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateLitestreamConfig(t *testing.T) {
	t.Run("should generate valid litestream config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "litestream.yml")

		err := GenerateLitestreamConfig("/data/app.db", "test-bucket", "", configPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read generated config: %v", err)
		}

		var config LitestreamConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			t.Fatalf("failed to parse generated YAML: %v", err)
		}

		if len(config.DBs) != 1 {
			t.Errorf("expected 1 db, got %d", len(config.DBs))
		}

		if config.DBs[0].Path != "/data/app.db" {
			t.Errorf("expected path /data/app.db, got %s", config.DBs[0].Path)
		}

		if len(config.DBs[0].Replicas) != 1 {
			t.Errorf("expected 1 replica, got %d", len(config.DBs[0].Replicas))
		}

		if config.DBs[0].Replicas[0].URL != "s3://test-bucket/app.db" {
			t.Errorf("expected URL s3://test-bucket/app.db, got %s", config.DBs[0].Replicas[0].URL)
		}
	})

	t.Run("should include custom S3 endpoint", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "litestream.yml")

		err := GenerateLitestreamConfig("/data/app.db", "test-bucket", "https://minio.local:9000", configPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		data, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read generated config: %v", err)
		}

		var config LitestreamConfig
		if err := yaml.Unmarshal(data, &config); err != nil {
			t.Fatalf("failed to parse generated YAML: %v", err)
		}

		if config.DBs[0].Replicas[0].Endpoint != "https://minio.local:9000" {
			t.Errorf("expected endpoint https://minio.local:9000, got %s", config.DBs[0].Replicas[0].Endpoint)
		}
	})

	t.Run("should create directory if not exists", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nested", "dir", "litestream.yml")

		err := GenerateLitestreamConfig("/data/app.db", "test-bucket", "", configPath)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("expected config file to exist at %s", configPath)
		}
	})
}
