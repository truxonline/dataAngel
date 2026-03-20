package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// generateLitestreamConfig creates a temporary litestream config file for restore
func generateLitestreamConfig(dbPath, bucket, s3Endpoint string) (string, error) {
	dbName := filepath.Base(dbPath)
	s3URL := fmt.Sprintf("s3://%s/%s", bucket, dbName)

	configContent := fmt.Sprintf(`dbs:
  - path: %s
    replicas:
      - url: %s`, dbPath, s3URL)

	if s3Endpoint != "" {
		configContent += fmt.Sprintf(`
        endpoint: %s`, s3Endpoint)
	}

	configPath := fmt.Sprintf("/tmp/litestream-restore-%d.yml", os.Getpid())
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write litestream config: %w", err)
	}

	return configPath, nil
}

// isSQLiteHealthy checks if a SQLite database is not corrupted
func isSQLiteHealthy(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	if err != nil {
		return false
	}

	if info.Size() == 0 {
		return false
	}

	db, err := sql.Open("sqlite3", path+"?mode=ro")
	if err != nil {
		return false
	}
	defer db.Close()

	var result string
	err = db.QueryRow("PRAGMA integrity_check").Scan(&result)
	if err != nil {
		return false
	}

	return result == "ok"
}

// restoreSQLite restores a single SQLite database using litestream
func restoreSQLite(ctx context.Context, bucket, s3Endpoint, dbPath string, timeout time.Duration) error {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return nil
	}

	log.Printf("[dataangel] restore database=%s", dbPath)

	wasCorrupted := false
	if _, err := os.Stat(dbPath); err == nil {
		if !isSQLiteHealthy(dbPath) {
			log.Printf("[dataangel] WARNING: Database exists but is corrupted, removing: %s", dbPath)
			if err := os.Remove(dbPath); err != nil {
				return fmt.Errorf("failed to remove corrupted database: %w", err)
			}
			wasCorrupted = true
			log.Printf("[dataangel] Corrupted database removed, proceeding with restore")
		}
	}

	// Always generate a config file — litestream requires one and falls
	// back to /etc/litestream.yml which doesn't exist in our container.
	configPath, err := generateLitestreamConfig(dbPath, bucket, s3Endpoint)
	if err != nil {
		return err
	}
	defer os.Remove(configPath)

	args := []string{
		"restore",
		"-config", configPath,
		"-if-db-not-exists",
		"-if-replica-exists",
		dbPath,
	}

	restoreCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(restoreCtx, "litestream", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 15 * time.Second

	log.Printf("[dataangel] Running: litestream %s", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("litestream restore failed: %w", err)
	}

	// If DB was corrupted and restore didn't produce a file, the backup
	// didn't exist on S3 — fail to prevent silent data loss (issue #20).
	if wasCorrupted {
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			return fmt.Errorf("CRITICAL: corrupted database was removed but no S3 backup exists for %s — refusing to start with empty database", dbPath)
		}
	}

	log.Printf("[dataangel] SQLite restored successfully: %s", dbPath)
	return nil
}

// restoreFilesystem restores a single filesystem path using rclone
func restoreFilesystem(ctx context.Context, bucket, s3Endpoint, fsPath string, timeout time.Duration) error {
	fsPath = strings.TrimSpace(fsPath)
	if fsPath == "" {
		return nil
	}

	log.Printf("[dataangel] restore filesystem=%s", fsPath)

	remotePath := fmt.Sprintf(":s3:%s/%s", bucket, filepath.Base(fsPath))

	args := []string{
		"copy",
		remotePath,
		fsPath,
		"--s3-env-auth",
		"--exclude", "*.db*",
		"--exclude", ".*.db-litestream/**",
		"--timeout", "60s",
		"--contimeout", "15s",
	}

	if s3Endpoint != "" {
		args = append(args, "--s3-provider", "Minio", "--s3-endpoint", s3Endpoint)
	} else {
		args = append(args, "--s3-provider", "AWS")
	}

	restoreCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	cmd := exec.CommandContext(restoreCtx, "rclone", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	cmd.WaitDelay = 15 * time.Second

	log.Printf("[dataangel] Running: rclone %s", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone copy failed: %w", err)
	}

	log.Printf("[dataangel] Filesystem restored successfully: %s", fsPath)
	return nil
}

// RunRestore executes the restore phase (all SQLite DBs + all filesystem paths)
func RunRestore(ctx context.Context, config Config) error {
	// Restore all SQLite databases
	for _, dbPath := range config.SqlitePaths {
		if err := restoreSQLite(ctx, config.Bucket, config.S3Endpoint, dbPath, config.RestoreTimeout); err != nil {
			return fmt.Errorf("failed to restore SQLite %s: %w", dbPath, err)
		}
	}

	// Restore all filesystem paths
	for _, fsPath := range config.FsPaths {
		if err := restoreFilesystem(ctx, config.Bucket, config.S3Endpoint, fsPath, config.RestoreTimeout); err != nil {
			return fmt.Errorf("failed to restore filesystem %s: %w", fsPath, err)
		}
	}

	return nil
}
