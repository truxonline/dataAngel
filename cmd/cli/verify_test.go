package cli

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/charchess/dataAngel/pkg/s3"
)

func TestVerifyCommand_Parsing(t *testing.T) {
	args := []string{"verify", "--bucket", "myapp"}
	cmd, err := ParseCommand(args)
	if err != nil {
		t.Errorf("La commande devrait être parsée sans erreur: %v", err)
	}
	if cmd.Bucket != "myapp" {
		t.Errorf("Le bucket devrait être 'myapp', got '%s'", cmd.Bucket)
	}
}

func TestVerifyCommand_MissingBucket(t *testing.T) {
	args := []string{"verify"}
	_, err := ParseCommand(args)
	if err == nil {
		t.Errorf("Une commande sans bucket devrait échouer")
	}
}

func TestFormatBackupStatus_ValidBackups(t *testing.T) {
	status := "Healthy"
	result := FormatBackupStatus(status)
	if result != "Statut: Healthy" {
		t.Errorf("Le formatage devrait être 'Statut: Healthy', got '%s'", result)
	}
}

func TestFormatBackupStatus_NoBackups(t *testing.T) {
	status := "No backups found"
	result := FormatBackupStatus(status)
	if result != "Aucun backup trouvé" {
		t.Errorf("Le formatage devrait être 'Aucun backup trouvé', got '%s'", result)
	}
}

type mockS3VerifyClient struct {
	backups []s3.BackupInfo
	err     error
}

func (m *mockS3VerifyClient) ListBackups(ctx context.Context, bucket, path string) ([]s3.BackupInfo, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.backups, nil
}

func TestVerifyBackupState(t *testing.T) {
	t.Run("should return formatted backup list", func(t *testing.T) {
		backups := []s3.BackupInfo{
			{Name: "backup1.db", Size: 1024, LastModified: time.Now(), Checksum: "abc123", Path: "/backup/backup1.db"},
			{Name: "backup2.db", Size: 2048, LastModified: time.Now().Add(-1 * time.Hour), Checksum: "def456", Path: "/backup/backup2.db"},
		}
		client := &mockS3VerifyClient{backups: backups}

		result, err := VerifyBackupState(context.Background(), client, "test-bucket")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result == "" {
			t.Error("Expected non-empty result")
		}
	})

	t.Run("should return no backups found message", func(t *testing.T) {
		client := &mockS3VerifyClient{backups: []s3.BackupInfo{}}

		result, err := VerifyBackupState(context.Background(), client, "test-bucket")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if result != "Aucun backup trouvé" {
			t.Errorf("Expected 'Aucun backup trouvé', got '%s'", result)
		}
	})

	t.Run("should return error on S3 failure", func(t *testing.T) {
		client := &mockS3VerifyClient{err: errors.New("S3 error")}

		_, err := VerifyBackupState(context.Background(), client, "test-bucket")
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestFormatBackupList(t *testing.T) {
	t.Run("should format multiple backups", func(t *testing.T) {
		backups := []s3.BackupInfo{
			{Name: "backup1.db", Size: 1024, LastModified: time.Now(), Checksum: "abc123", Path: "/backup/backup1.db"},
			{Name: "backup2.db", Size: 2048, LastModified: time.Now().Add(-1 * time.Hour), Checksum: "def456", Path: "/backup/backup2.db"},
		}

		result := FormatBackupList(backups)
		if result == "" {
			t.Error("Expected non-empty result")
		}
		// Check if result contains backup names
		if !contains(result, "backup1.db") || !contains(result, "backup2.db") {
			t.Errorf("Result should contain backup names, got: %s", result)
		}
	})

	t.Run("should format single backup", func(t *testing.T) {
		backups := []s3.BackupInfo{
			{Name: "backup1.db", Size: 1024, LastModified: time.Now(), Checksum: "abc123", Path: "/backup/backup1.db"},
		}

		result := FormatBackupList(backups)
		if result == "" {
			t.Error("Expected non-empty result")
		}
		if !contains(result, "backup1.db") {
			t.Errorf("Result should contain backup name, got: %s", result)
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
