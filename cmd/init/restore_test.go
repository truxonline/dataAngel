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
	tests := []struct {
		name        string
		dbPath      string
		bucket      string
		s3Endpoint  string
		wantContent []string
	}{
		{
			name:       "with custom endpoint",
			dbPath:     "/data/test.db",
			bucket:     "my-bucket",
			s3Endpoint: "http://minio.minio.svc.cluster.local:9000",
			wantContent: []string{
				"dbs:",
				"  - path: /data/test.db",
				"    replicas:",
				"      - url: s3://my-bucket/test.db",
				"        endpoint: http://minio.minio.svc.cluster.local:9000",
			},
		},
		{
			name:       "without custom endpoint",
			dbPath:     "/app/data/mealie.db",
			bucket:     "backups",
			s3Endpoint: "",
			wantContent: []string{
				"dbs:",
				"  - path: /app/data/mealie.db",
				"    replicas:",
				"      - url: s3://backups/mealie.db",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configPath, err := generateLitestreamConfig(tt.dbPath, tt.bucket, tt.s3Endpoint)
			if err != nil {
				t.Fatalf("generateLitestreamConfig() error = %v", err)
			}
			defer os.Remove(configPath)

			content, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatalf("failed to read config file: %v", err)
			}

			contentStr := string(content)
			for _, want := range tt.wantContent {
				if !strings.Contains(contentStr, want) {
					t.Errorf("config missing expected content.\nwant substring: %q\ngot:\n%s", want, contentStr)
				}
			}

			if tt.s3Endpoint == "" && strings.Contains(contentStr, "endpoint:") {
				t.Error("config should not contain endpoint field when s3Endpoint is empty")
			}
		})
	}
}

func TestGenerateLitestreamConfig_FileCreation(t *testing.T) {
	configPath, err := generateLitestreamConfig("/data/test.db", "bucket", "http://minio:9000")
	if err != nil {
		t.Fatalf("generateLitestreamConfig() error = %v", err)
	}
	defer os.Remove(configPath)

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("config file was not created")
	}

	if !strings.HasPrefix(configPath, "/tmp/litestream-restore-") {
		t.Errorf("unexpected config path: %s", configPath)
	}

	if !strings.HasSuffix(configPath, ".yml") {
		t.Errorf("config path should end with .yml: %s", configPath)
	}
}

func TestIsSQLiteHealthy(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(t *testing.T) string
		expected bool
	}{
		{
			name: "healthy database",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				dbPath := filepath.Join(tmpDir, "healthy.db")

				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					t.Fatalf("failed to create test DB: %v", err)
				}
				defer db.Close()

				_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY, name TEXT)")
				if err != nil {
					t.Fatalf("failed to create table: %v", err)
				}

				_, err = db.Exec("INSERT INTO test (name) VALUES ('test')")
				if err != nil {
					t.Fatalf("failed to insert data: %v", err)
				}

				return dbPath
			},
			expected: true,
		},
		{
			name: "corrupted database header",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				dbPath := filepath.Join(tmpDir, "corrupted.db")

				db, err := sql.Open("sqlite3", dbPath)
				if err != nil {
					t.Fatalf("failed to create test DB: %v", err)
				}
				defer db.Close()

				_, err = db.Exec("CREATE TABLE test (id INTEGER PRIMARY KEY)")
				if err != nil {
					t.Fatalf("failed to create table: %v", err)
				}
				db.Close()

				file, err := os.OpenFile(dbPath, os.O_RDWR, 0644)
				if err != nil {
					t.Fatalf("failed to open DB for corruption: %v", err)
				}
				defer file.Close()

				_, err = file.WriteAt([]byte("CORRUPTED_HEADER"), 0)
				if err != nil {
					t.Fatalf("failed to corrupt header: %v", err)
				}

				return dbPath
			},
			expected: false,
		},
		{
			name: "non-existent database",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nonexistent.db")
			},
			expected: false,
		},
		{
			name: "empty file",
			setup: func(t *testing.T) string {
				tmpDir := t.TempDir()
				dbPath := filepath.Join(tmpDir, "empty.db")

				if err := os.WriteFile(dbPath, []byte{}, 0644); err != nil {
					t.Fatalf("failed to create empty file: %v", err)
				}

				return dbPath
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dbPath := tt.setup(t)
			result := isSQLiteHealthy(dbPath)

			if result != tt.expected {
				t.Errorf("isSQLiteHealthy() = %v, want %v", result, tt.expected)
			}
		})
	}
}
