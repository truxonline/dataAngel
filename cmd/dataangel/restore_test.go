package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateLitestreamConfig(t *testing.T) {
	t.Run("with custom S3 endpoint", func(t *testing.T) {
		configPath, err := generateLitestreamConfig("/app/data/mealie.db", "my-bucket", "http://minio:9000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(configPath)

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}

		s := string(content)
		if !strings.Contains(s, "path: /app/data/mealie.db") {
			t.Error("config should contain database path")
		}
		if !strings.Contains(s, "url: s3://my-bucket/mealie.db") {
			t.Error("config should contain S3 URL")
		}
		if !strings.Contains(s, "endpoint: http://minio:9000") {
			t.Error("config should contain custom endpoint")
		}
	})

	t.Run("without custom S3 endpoint", func(t *testing.T) {
		configPath, err := generateLitestreamConfig("/app/data/app.db", "prod-bucket", "")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(configPath)

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}

		s := string(content)
		if !strings.Contains(s, "path: /app/data/app.db") {
			t.Error("config should contain database path")
		}
		if !strings.Contains(s, "url: s3://prod-bucket/app.db") {
			t.Error("config should contain S3 URL")
		}
		if strings.Contains(s, "endpoint:") {
			t.Error("config should NOT contain endpoint when s3Endpoint is empty")
		}
	})

	t.Run("config file is valid YAML", func(t *testing.T) {
		configPath, err := generateLitestreamConfig("/data/test.db", "bucket", "http://localhost:9000")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		defer os.Remove(configPath)

		content, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("failed to read config: %v", err)
		}

		// Must start with "dbs:" — basic structure check
		if !strings.HasPrefix(string(content), "dbs:") {
			t.Error("config should start with 'dbs:'")
		}
	})
}

func TestIsSQLiteHealthy(t *testing.T) {
	t.Run("nonexistent file returns false", func(t *testing.T) {
		if isSQLiteHealthy("/nonexistent/path.db") {
			t.Error("nonexistent file should not be healthy")
		}
	})

	t.Run("empty file returns false", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "empty.db")
		os.WriteFile(tmp, []byte{}, 0644)
		if isSQLiteHealthy(tmp) {
			t.Error("empty file should not be healthy")
		}
	})

	t.Run("corrupted file returns false", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "corrupt.db")
		os.WriteFile(tmp, []byte("this is not a sqlite database"), 0644)
		if isSQLiteHealthy(tmp) {
			t.Error("corrupted file should not be healthy")
		}
	})
}

func TestRestoreSQLiteSkipsEmpty(t *testing.T) {
	t.Run("empty dbPath is skipped", func(t *testing.T) {
		err := restoreSQLite(nil, "bucket", "", "", 0)
		if err != nil {
			t.Errorf("empty dbPath should return nil, got: %v", err)
		}
	})

	t.Run("whitespace dbPath is skipped", func(t *testing.T) {
		err := restoreSQLite(nil, "bucket", "", "   ", 0)
		if err != nil {
			t.Errorf("whitespace dbPath should return nil, got: %v", err)
		}
	})
}

func TestRestoreFilesystemSkipsEmpty(t *testing.T) {
	t.Run("empty fsPath is skipped", func(t *testing.T) {
		err := restoreFilesystem(nil, "bucket", "", "", 0)
		if err != nil {
			t.Errorf("empty fsPath should return nil, got: %v", err)
		}
	})
}
