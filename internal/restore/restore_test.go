package restore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestShouldSkip(t *testing.T) {
	t.Run("should skip when local is newer", func(t *testing.T) {
		decision := DecisionSkip
		result := ShouldSkip(decision)
		if !result {
			t.Errorf("Expected ShouldSkip to return true for DecisionSkip, got false")
		}
	})

	t.Run("should not skip when restore is needed", func(t *testing.T) {
		decision := DecisionRestore
		result := ShouldSkip(decision)
		if result {
			t.Errorf("Expected ShouldSkip to return false for DecisionRestore, got true")
		}
	})

	t.Run("should not skip when data is corrupted", func(t *testing.T) {
		decision := DecisionCorrupted
		result := ShouldSkip(decision)
		if result {
			t.Errorf("Expected ShouldSkip to return false for DecisionCorrupted, got true")
		}
	})
}

type mockS3Downloader struct {
	err         error
	corruptData bool
}

func (m *mockS3Downloader) Download(ctx context.Context, bucket, key, destPath string) error {
	if m.err != nil {
		return m.err
	}
	if m.corruptData {
		// Write corrupted data (wrong checksum)
		return os.WriteFile(destPath, []byte("corrupted data"), 0644)
	}
	// Write valid data
	return os.WriteFile(destPath, []byte("valid data"), 0644)
}

func TestRestoreFromS3(t *testing.T) {
	t.Run("should restore successfully", func(t *testing.T) {
		tmpDir := t.TempDir()
		destPath := filepath.Join(tmpDir, "data.db")
		downloader := &mockS3Downloader{}
		config := RestoreConfig{
			Bucket:           "test-bucket",
			Key:              "backup/data.db",
			DestPath:         destPath,
			ExpectedChecksum: "d63e23e8a7cbe080f2a79984fb4b2e08d22924e0f27fa7b30220e4e351489962", // SHA256 of "valid data"
		}

		err := RestoreFromS3(context.Background(), downloader, config)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("should return error on download failure", func(t *testing.T) {
		tmpDir := t.TempDir()
		destPath := filepath.Join(tmpDir, "data.db")
		downloader := &mockS3Downloader{err: errors.New("S3 error")}
		config := RestoreConfig{
			Bucket:           "test-bucket",
			Key:              "backup/data.db",
			DestPath:         destPath,
			ExpectedChecksum: "a4b2c8d6e0f2a4b2c8d6e0f2a4b2c8d6e0f2a4b2c8d6e0f2a4b2c8d6e0f2a4b2",
		}

		err := RestoreFromS3(context.Background(), downloader, config)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("should return error on corrupt download", func(t *testing.T) {
		tmpDir := t.TempDir()
		destPath := filepath.Join(tmpDir, "data.db")
		downloader := &mockS3Downloader{corruptData: true}
		config := RestoreConfig{
			Bucket:           "test-bucket",
			Key:              "backup/data.db",
			DestPath:         destPath,
			ExpectedChecksum: "d63e23e8a7cbe080f2a79984fb4b2e08d22924e0f27fa7b30220e4e351489962", // Different from corrupted data
		}

		err := RestoreFromS3(context.Background(), downloader, config)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}

func TestVerifyRestoredData(t *testing.T) {
	t.Run("should verify valid data", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "data.db")
		content := []byte("valid data")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
		// SHA256 of "valid data"
		expectedChecksum := "d63e23e8a7cbe080f2a79984fb4b2e08d22924e0f27fa7b30220e4e351489962"

		err := VerifyRestoredData(path, expectedChecksum)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("should fail on invalid checksum", func(t *testing.T) {
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "data.db")
		content := []byte("corrupted data")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
		// Different checksum (SHA256 of "corrupted data")
		expectedChecksum := "d63e23e8a7cbe080f2a79984fb4b2e08d22924e0f27fa7b30220e4e351489962"

		err := VerifyRestoredData(path, expectedChecksum)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})
}
