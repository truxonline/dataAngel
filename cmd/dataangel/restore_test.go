package main

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
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

func TestIsSQLiteQuickCheck(t *testing.T) {
	t.Run("nonexistent file returns false", func(t *testing.T) {
		if isSQLiteQuickCheck("/nonexistent/path.db") {
			t.Error("nonexistent file should not pass quick check")
		}
	})

	t.Run("empty file returns false", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "empty.db")
		os.WriteFile(tmp, []byte{}, 0644)
		if isSQLiteQuickCheck(tmp) {
			t.Error("empty file should not pass quick check")
		}
	})

	t.Run("corrupted file returns false", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "corrupt.db")
		os.WriteFile(tmp, []byte("this is not a sqlite database"), 0644)
		if isSQLiteQuickCheck(tmp) {
			t.Error("corrupted file should not pass quick check")
		}
	})

	t.Run("valid SQLite DB returns true", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "valid.db")
		db, err := sql.Open("sqlite3", tmp)
		if err != nil {
			t.Fatalf("failed to create test DB: %v", err)
		}
		db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
		db.Close()

		if !isSQLiteQuickCheck(tmp) {
			t.Error("valid SQLite DB should pass quick check")
		}
	})
}

func TestIsSQLiteHealthy(t *testing.T) {
	t.Run("nonexistent file returns false", func(t *testing.T) {
		if isSQLiteHealthy("/nonexistent/path.db") {
			t.Error("nonexistent file should not be healthy")
		}
	})

	t.Run("valid SQLite DB returns true", func(t *testing.T) {
		tmp := filepath.Join(t.TempDir(), "valid.db")
		db, err := sql.Open("sqlite3", tmp)
		if err != nil {
			t.Fatalf("failed to create test DB: %v", err)
		}
		db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
		db.Close()

		if !isSQLiteHealthy(tmp) {
			t.Error("valid SQLite DB should be healthy")
		}
	})
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{2411724800, "2.2 GB"},
	}
	for _, tt := range tests {
		if got := formatSize(tt.bytes); got != tt.expected {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.expected)
		}
	}
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
		err := restoreFilesystem(nil, "bucket", "", "", 0, nil, false)
		if err != nil {
			t.Errorf("empty fsPath should return nil, got: %v", err)
		}
	})
}

func TestBuildRcloneRestoreArgs(t *testing.T) {
	t.Run("--update present by default", func(t *testing.T) {
		args := buildRcloneRestoreArgs("my-bucket", "", "/config", nil, false)
		found := false
		for _, a := range args {
			if a == "--update" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected --update in args when restoreOverwrite=false, got: %v", args)
		}
	})

	t.Run("--update absent when restoreOverwrite=true", func(t *testing.T) {
		args := buildRcloneRestoreArgs("my-bucket", "", "/config", nil, true)
		for _, a := range args {
			if a == "--update" {
				t.Errorf("expected no --update in args when restoreOverwrite=true, got: %v", args)
			}
		}
	})

	t.Run("S3 path uses filepath.Base", func(t *testing.T) {
		args := buildRcloneRestoreArgs("bucket", "", "/home/app/config", nil, false)
		if args[1] != ":s3:bucket/config" {
			t.Errorf("expected S3 remote ':s3:bucket/config', got %q", args[1])
		}
	})

	t.Run("MinIO endpoint adds provider flags", func(t *testing.T) {
		args := buildRcloneRestoreArgs("bucket", "http://minio:9000", "/data", nil, false)
		hasProvider, hasEndpoint := false, false
		for i, a := range args {
			if a == "--s3-provider" && i+1 < len(args) && args[i+1] == "Minio" {
				hasProvider = true
			}
			if a == "--s3-endpoint" && i+1 < len(args) && args[i+1] == "http://minio:9000" {
				hasEndpoint = true
			}
		}
		if !hasProvider || !hasEndpoint {
			t.Errorf("expected MinIO provider/endpoint flags, got: %v", args)
		}
	})

	t.Run("exclude patterns are forwarded", func(t *testing.T) {
		args := buildRcloneRestoreArgs("bucket", "", "/data", []string{"*.log", "*.tmp"}, false)
		var excludes []string
		for i, a := range args {
			if a == "--exclude" && i+1 < len(args) {
				excludes = append(excludes, args[i+1])
			}
		}
		if len(excludes) != 2 || excludes[0] != "*.log" || excludes[1] != "*.tmp" {
			t.Errorf("unexpected exclude patterns: %v", excludes)
		}
	})
}
