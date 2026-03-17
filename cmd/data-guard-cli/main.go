package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/charchess/dataAngel/cmd/cli"
	"github.com/charchess/dataAngel/pkg/s3"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: data-guard-cli <command> [args]")
		fmt.Fprintln(os.Stderr, "Commands: verify, force-release-lock")
		os.Exit(1)
	}

	command := os.Args[1]
	switch command {
	case "verify":
		cmd, err := cli.ParseCommand(os.Args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing command: %v\n", err)
			os.Exit(1)
		}

		// In a real implementation, this would connect to S3
		// For now, we'll use a mock client
		client := &mockS3VerifyClient{}
		result, err := cli.VerifyBackupState(context.Background(), client, cmd.Bucket)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error verifying backup state: %v\n", err)
			os.Exit(1)
		}

		fmt.Println(result)

	case "force-release-lock":
		cmd, err := cli.ParseForceReleaseCommand(os.Args[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing command: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Force releasing lock: %s\n", cmd.LockID)

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		fmt.Fprintln(os.Stderr, "Available commands: verify, force-release-lock")
		os.Exit(1)
	}
}

// mockS3VerifyClient is a placeholder implementation for testing
type mockS3VerifyClient struct{}

func (m *mockS3VerifyClient) ListBackups(ctx context.Context, bucket, path string) ([]s3.BackupInfo, error) {
	// Return mock backups
	return []s3.BackupInfo{
		{Name: "backup1.db", Size: 1024, LastModified: time.Now(), Checksum: "abc123", Path: "/backup/backup1.db"},
		{Name: "backup2.db", Size: 2048, LastModified: time.Now().Add(-1 * time.Hour), Checksum: "def456", Path: "/backup/backup2.db"},
	}, nil
}
