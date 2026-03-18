package main

import (
	"os"
	"strings"
	"testing"
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
