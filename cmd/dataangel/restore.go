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

// isSQLiteQuickCheck does a fast validation: file exists, non-zero, valid
// SQLite header, and PRAGMA quick_check (checks B-tree structure without
// reading every page). Use this for the common "DB already exists on PVC"
// path to avoid the 30+ minute full integrity_check on large DBs (#39).
func isSQLiteQuickCheck(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
		return false
	}

	db, err := sql.Open("sqlite3", path+"?mode=ro")
	if err != nil {
		return false
	}
	defer db.Close()

	var result string
	err = db.QueryRow("PRAGMA quick_check").Scan(&result)
	if err != nil {
		return false
	}

	return result == "ok"
}

// isSQLiteHealthy does a full PRAGMA integrity_check. This reads every page
// and can take 30+ minutes on large databases. Only use after restoring
// from S3 to verify the backup isn't corrupted.
func isSQLiteHealthy(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.Size() == 0 {
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

// formatSize returns a human-readable file size.
func formatSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.1f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.1f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.1f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}

// cleanShutdownSentinel returns the sentinel file path for a given DB path.
func cleanShutdownSentinel(dbPath string) string {
	return dbPath + ".dataangel-clean"
}

// writeCleanShutdownSentinels writes sentinel files for all SQLite paths,
// indicating the previous shutdown was graceful (issue #42).
func writeCleanShutdownSentinels(sqlitePaths []string) {
	for _, dbPath := range sqlitePaths {
		sentinel := cleanShutdownSentinel(dbPath)
		if err := os.WriteFile(sentinel, []byte("clean"), 0644); err != nil {
			log.Printf("[dataangel] WARNING: failed to write clean shutdown sentinel: %v", err)
		}
	}
}

// logFileProgress logs the size of a file periodically while it's being
// written (e.g. during litestream restore). Stops when ctx is cancelled.
func logFileProgress(ctx context.Context, path string, operation string, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	start := time.Now()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			info, err := os.Stat(path)
			if err != nil {
				continue
			}
			log.Printf("[dataangel] %s progress: %s (%s, %v elapsed)", operation, path, formatSize(info.Size()), time.Since(start).Round(time.Second))
		}
	}
}

// restoreSQLite restores a single SQLite database using litestream
func restoreSQLite(ctx context.Context, bucket, s3Endpoint, dbPath string, timeout time.Duration) error {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return nil
	}

	start := time.Now()

	// Check if DB already exists on disk
	log.Printf("[dataangel] checking database: %s", dbPath)
	info, statErr := os.Stat(dbPath)
	dbExisted := statErr == nil

	if dbExisted {
		log.Printf("[dataangel] database found on disk: %s (size: %s)", dbPath, formatSize(info.Size()))

		// Fastest path: if previous shutdown was clean, skip validation
		// entirely. The sentinel file is written on SIGTERM (issue #42).
		sentinel := cleanShutdownSentinel(dbPath)
		if _, err := os.Stat(sentinel); err == nil {
			os.Remove(sentinel)
			log.Printf("[dataangel] clean shutdown detected, skipping validation: %s", dbPath)
			return nil
		}

		// Fast path: DB exists, do a quick check instead of full integrity
		// check. PRAGMA quick_check validates B-tree structure without
		// reading every page — seconds vs 30+ min on large DBs (#39).
		log.Printf("[dataangel] no clean shutdown sentinel, running quick validation (PRAGMA quick_check)...")
		checkStart := time.Now()
		if isSQLiteQuickCheck(dbPath) {
			log.Printf("[dataangel] database valid, skipping restore: %s (validated in %v)", dbPath, time.Since(checkStart).Round(time.Millisecond))
			return nil
		}

		// Quick check failed — DB is corrupted
		log.Printf("[dataangel] WARNING: database exists but failed quick_check (took %v), removing: %s", time.Since(checkStart).Round(time.Millisecond), dbPath)
		if err := os.Remove(dbPath); err != nil {
			return fmt.Errorf("failed to remove corrupted database: %w", err)
		}
		log.Printf("[dataangel] corrupted database removed, proceeding with S3 restore")
	} else {
		log.Printf("[dataangel] database not found on disk: %s", dbPath)
	}

	// DB doesn't exist (or was removed) — attempt litestream restore from S3
	log.Printf("[dataangel] preparing litestream restore from S3 (timeout: %v)...", timeout)
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

	log.Printf("[dataangel] running: litestream %s", strings.Join(args, " "))
	litestreamStart := time.Now()

	// Monitor file size during restore for progress reporting
	progressCtx, progressCancel := context.WithCancel(restoreCtx)
	go logFileProgress(progressCtx, dbPath, "litestream restore", 5*time.Second)

	if err := cmd.Run(); err != nil {
		progressCancel()
		return fmt.Errorf("litestream restore failed: %w", err)
	}
	progressCancel()
	log.Printf("[dataangel] litestream completed in %v", time.Since(litestreamStart).Round(time.Millisecond))

	// Check what actually happened after litestream exited 0.
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		if dbExisted {
			// Corrupted DB was removed but no S3 backup exists (issue #20).
			return fmt.Errorf("CRITICAL: corrupted database was removed but no S3 backup exists for %s — refusing to start with empty database", dbPath)
		}
		// First deployment: no backup yet, app will create fresh DB (issue #27).
		log.Printf("[dataangel] no S3 backup found for %s, app will create fresh database", dbPath)
		return nil
	}

	// DB was restored from S3 — do full integrity check (issue #26).
	restoredInfo, _ := os.Stat(dbPath)
	log.Printf("[dataangel] database restored from S3: %s (size: %s), running full integrity check...", dbPath, formatSize(restoredInfo.Size()))
	checkStart := time.Now()
	if !isSQLiteHealthy(dbPath) {
		return fmt.Errorf("CRITICAL: restored database failed integrity check: %s — S3 backup may be corrupted", dbPath)
	}

	log.Printf("[dataangel] SQLite restored and verified: %s (integrity check: %v, total: %v)", dbPath, time.Since(checkStart).Round(time.Millisecond), time.Since(start).Round(time.Millisecond))
	return nil
}

// restoreFilesystem restores a single filesystem path using rclone
func restoreFilesystem(ctx context.Context, bucket, s3Endpoint, fsPath string, timeout time.Duration, excludePatterns []string) error {
	fsPath = strings.TrimSpace(fsPath)
	if fsPath == "" {
		return nil
	}

	start := time.Now()
	log.Printf("[dataangel] restore filesystem=%s (timeout: %v)", fsPath, timeout)

	remotePath := fmt.Sprintf(":s3:%s/%s", bucket, filepath.Base(fsPath))

	args := []string{
		"copy",
		remotePath,
		fsPath,
		"--s3-env-auth",
		"--timeout", "60s",
		"--contimeout", "15s",
		"--stats", "5s",
		"--stats-one-line",
	}

	for _, pattern := range excludePatterns {
		args = append(args, "--exclude", pattern)
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

	log.Printf("[dataangel] Filesystem restored: %s (elapsed: %v)", fsPath, time.Since(start).Round(time.Millisecond))
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
		if err := restoreFilesystem(ctx, config.Bucket, config.S3Endpoint, fsPath, config.RestoreTimeout, config.ExcludePatterns); err != nil {
			return fmt.Errorf("failed to restore filesystem %s: %w", fsPath, err)
		}
	}

	return nil
}
