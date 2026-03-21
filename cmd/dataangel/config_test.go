package main

import (
	"os"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Save and restore environment
	envVars := []string{
		"DATA_GUARD_BUCKET", "DATA_GUARD_SQLITE_PATHS", "DATA_GUARD_FS_PATHS",
		"DATA_GUARD_DEPLOYMENT_NAME", "DATA_GUARD_S3_ENDPOINT",
		"DATA_GUARD_RESTORE_TIMEOUT", "DATA_GUARD_RCLONE_INTERVAL",
		"DATA_GUARD_SHUTDOWN_TIMEOUT", "DATA_GUARD_LOCK_TTL",
		"DATA_GUARD_METRICS_ENABLED", "DATA_GUARD_METRICS_PORT",
		"DATA_GUARD_FULL_LOGS", "DATA_GUARD_YAML_PATHS",
		"DATA_GUARD_EXCLUDE_PATTERNS", "DATA_GUARD_SYNC_TIMEOUT",
		"DATA_GUARD_LOCK_ACQUIRE_TIMEOUT", "DATA_GUARD_RCLONE_DELAY",
		"DATA_GUARD_RCLONE_TRANSFERS", "DATA_GUARD_RCLONE_CHECKERS",
		"DATA_GUARD_RCLONE_BWLIMIT",
	}
	saved := make(map[string]string)
	for _, k := range envVars {
		saved[k] = os.Getenv(k)
	}
	t.Cleanup(func() {
		for k, v := range saved {
			if v == "" {
				os.Unsetenv(k)
			} else {
				os.Setenv(k, v)
			}
		}
	})

	clearEnv := func() {
		for _, k := range envVars {
			os.Unsetenv(k)
		}
	}

	t.Run("missing bucket returns error", func(t *testing.T) {
		clearEnv()
		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for missing bucket")
		}
	})

	t.Run("missing paths returns error", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "test-bucket")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "test")
		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for missing paths")
		}
	})

	t.Run("missing deployment name returns error", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "test-bucket")
		os.Setenv("DATA_GUARD_SQLITE_PATHS", "/app/data/test.db")
		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for missing deployment name")
		}
	})

	t.Run("valid minimal config", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "test-bucket")
		os.Setenv("DATA_GUARD_SQLITE_PATHS", "/app/data/test.db")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "mealie")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.Bucket != "test-bucket" {
			t.Errorf("expected bucket 'test-bucket', got '%s'", cfg.Bucket)
		}
		if len(cfg.SqlitePaths) != 1 || cfg.SqlitePaths[0] != "/app/data/test.db" {
			t.Errorf("unexpected sqlite paths: %v", cfg.SqlitePaths)
		}
		if cfg.RestoreTimeout.Minutes() != 10 {
			t.Errorf("expected default restore timeout 10m, got %v", cfg.RestoreTimeout)
		}
		if cfg.LockTTL.Seconds() != 60 {
			t.Errorf("expected default lock TTL 60s, got %v", cfg.LockTTL)
		}
		if cfg.RcloneInterval.Seconds() != 60 {
			t.Errorf("expected default rclone interval 60s, got %v", cfg.RcloneInterval)
		}
		if !cfg.MetricsEnabled {
			t.Error("metrics should be enabled by default")
		}
		if cfg.MetricsPort != 9090 {
			t.Errorf("expected default metrics port 9090, got %d", cfg.MetricsPort)
		}
	})

	t.Run("custom restore timeout", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "b")
		os.Setenv("DATA_GUARD_SQLITE_PATHS", "/db")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "d")
		os.Setenv("DATA_GUARD_RESTORE_TIMEOUT", "5m")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.RestoreTimeout.Minutes() != 5 {
			t.Errorf("expected restore timeout 5m, got %v", cfg.RestoreTimeout)
		}
	})

	t.Run("invalid restore timeout returns error", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "b")
		os.Setenv("DATA_GUARD_SQLITE_PATHS", "/db")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "d")
		os.Setenv("DATA_GUARD_RESTORE_TIMEOUT", "not-a-duration")

		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for invalid restore timeout")
		}
	})

	t.Run("multiple sqlite and fs paths", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "b")
		os.Setenv("DATA_GUARD_SQLITE_PATHS", "/db1,/db2")
		os.Setenv("DATA_GUARD_FS_PATHS", "/data,/config")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "d")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.SqlitePaths) != 2 {
			t.Errorf("expected 2 sqlite paths, got %d", len(cfg.SqlitePaths))
		}
		if len(cfg.FsPaths) != 2 {
			t.Errorf("expected 2 fs paths, got %d", len(cfg.FsPaths))
		}
	})

	t.Run("S3 prefix collision in sqlite paths returns error", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "b")
		os.Setenv("DATA_GUARD_SQLITE_PATHS", "/db1/app.db,/db2/app.db")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "d")

		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for S3 prefix collision")
		}
	})

	t.Run("S3 prefix collision in fs paths returns error", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "b")
		os.Setenv("DATA_GUARD_FS_PATHS", "/volume1/data,/volume2/data")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "d")

		_, err := LoadConfig()
		if err == nil {
			t.Fatal("expected error for S3 prefix collision")
		}
	})

	t.Run("custom exclude patterns", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "b")
		os.Setenv("DATA_GUARD_SQLITE_PATHS", "/db")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "d")
		os.Setenv("DATA_GUARD_EXCLUDE_PATTERNS", "*.log,*.tmp")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.ExcludePatterns) != 2 || cfg.ExcludePatterns[0] != "*.log" {
			t.Errorf("unexpected exclude patterns: %v", cfg.ExcludePatterns)
		}
	})

	t.Run("default exclude patterns", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "b")
		os.Setenv("DATA_GUARD_SQLITE_PATHS", "/db")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "d")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(cfg.ExcludePatterns) != 2 || cfg.ExcludePatterns[0] != "*.db*" {
			t.Errorf("unexpected default exclude patterns: %v", cfg.ExcludePatterns)
		}
	})

	t.Run("configurable sync timeout and lock acquire timeout", func(t *testing.T) {
		clearEnv()
		os.Setenv("DATA_GUARD_BUCKET", "b")
		os.Setenv("DATA_GUARD_SQLITE_PATHS", "/db")
		os.Setenv("DATA_GUARD_DEPLOYMENT_NAME", "d")
		os.Setenv("DATA_GUARD_SYNC_TIMEOUT", "5m")
		os.Setenv("DATA_GUARD_LOCK_ACQUIRE_TIMEOUT", "10m")

		cfg, err := LoadConfig()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if cfg.SyncTimeout.Minutes() != 5 {
			t.Errorf("expected sync timeout 5m, got %v", cfg.SyncTimeout)
		}
		if cfg.LockAcquireTimeout.Minutes() != 10 {
			t.Errorf("expected lock acquire timeout 10m, got %v", cfg.LockAcquireTimeout)
		}
	})
}
