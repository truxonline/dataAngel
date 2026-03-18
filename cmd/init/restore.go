package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func restoreSQLite(ctx context.Context, bucket, s3Endpoint, dbPath string) error {
	dbPath = strings.TrimSpace(dbPath)
	if dbPath == "" {
		return nil
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
