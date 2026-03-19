package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

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

func restoreSQLite(ctx context.Context, bucket, s3Endpoint, dbPath string) error {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return nil
	}

	if _, err := os.Stat(dbPath); err == nil {
		if !isSQLiteHealthy(dbPath) {
			fmt.Printf("WARNING: Database exists but is corrupted, removing: %s\n", dbPath)
			if err := os.Remove(dbPath); err != nil {
				return fmt.Errorf("failed to remove corrupted database: %w", err)
			}
			fmt.Printf("Corrupted database removed, proceeding with restore\n")
		}
	}

	var args []string
	var configPath string

	if s3Endpoint != "" {
		var err error
		configPath, err = generateLitestreamConfig(dbPath, bucket, s3Endpoint)
		if err != nil {
			return err
		}
		defer os.Remove(configPath)

		args = []string{
			"restore",
			"-config", configPath,
			"-if-db-not-exists",
			"-if-replica-exists",
			dbPath,
		}
	} else {
		dbName := filepath.Base(dbPath)
		replicaURL := fmt.Sprintf("s3://%s/%s", bucket, dbName)

		args = []string{
			"restore",
			"-if-db-not-exists",
			"-if-replica-exists",
			"-replica", replicaURL,
			dbPath,
		}
	}

	cmd := exec.CommandContext(ctx, "litestream", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	fmt.Printf("Running: litestream %s\n", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("litestream restore failed: %w", err)
	}

	fmt.Printf("SQLite restored successfully: %s\n", dbPath)
	return nil
}

func restoreFilesystem(ctx context.Context, bucket, s3Endpoint, fsPath string) error {
	fsPath = strings.TrimSpace(fsPath)
	if fsPath == "" {
		return nil
	}

	remotePath := fmt.Sprintf(":s3:%s/%s", bucket, filepath.Base(fsPath))

	args := []string{
		"copy",
		remotePath,
		fsPath,
		"--s3-env-auth",
		"--exclude", "*.db*",
	}

	if s3Endpoint != "" {
		args = append(args, "--s3-provider", "Minio", "--s3-endpoint", s3Endpoint)
	} else {
		args = append(args, "--s3-provider", "AWS")
	}

	cmd := exec.CommandContext(ctx, "rclone", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	fmt.Printf("Running: rclone %s\n", strings.Join(args, " "))

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("rclone copy failed: %w", err)
	}

	fmt.Printf("Filesystem restored successfully: %s\n", fsPath)
	return nil
}
