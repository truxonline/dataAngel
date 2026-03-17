package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/charchess/dataAngel/pkg/s3"
)

type VerifyCommand struct {
	Bucket string
}

func ParseCommand(args []string) (*VerifyCommand, error) {
	if len(args) < 2 {
		return nil, fmt.Errorf("manque d'arguments")
	}

	for i, arg := range args {
		if arg == "--bucket" && i+1 < len(args) {
			return &VerifyCommand{Bucket: args[i+1]}, nil
		}
		if strings.HasPrefix(arg, "--bucket=") {
			bucket := strings.TrimPrefix(arg, "--bucket=")
			return &VerifyCommand{Bucket: bucket}, nil
		}
	}

	return nil, fmt.Errorf("option --bucket manquante")
}

func FormatBackupStatus(status string) string {
	if status == "" || status == "No backups found" {
		return "Aucun backup trouvé"
	}
	return fmt.Sprintf("Statut: %s", status)
}

// S3VerifyClient defines the interface for listing backups from S3.
type S3VerifyClient interface {
	ListBackups(ctx context.Context, bucket, path string) ([]s3.BackupInfo, error)
}

// VerifyBackupState checks the state of backups in S3 and returns a formatted string.
func VerifyBackupState(ctx context.Context, client S3VerifyClient, bucket string) (string, error) {
	backups, err := client.ListBackups(ctx, bucket, "")
	if err != nil {
		return "", fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		return FormatBackupStatus("No backups found"), nil
	}

	return FormatBackupList(backups), nil
}

// FormatBackupList formats a list of backups into a human-readable string.
func FormatBackupList(backups []s3.BackupInfo) string {
	if len(backups) == 0 {
		return FormatBackupStatus("No backups found")
	}

	var builder strings.Builder
	builder.WriteString("Backups found:\n")
	for _, backup := range backups {
		builder.WriteString(fmt.Sprintf("  - %s (Size: %d, Modified: %s)\n",
			backup.Name, backup.Size, backup.LastModified.Format("2006-01-02 15:04:05")))
	}
	return builder.String()
}
