package restore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// RestoreConfig contains configuration for restoring data from S3.
type RestoreConfig struct {
	Bucket           string
	Key              string
	DestPath         string
	ExpectedChecksum string
}

// S3Downloader defines the interface for downloading files from S3.
type S3Downloader interface {
	Download(ctx context.Context, bucket, key, destPath string) error
}

// RestoreFromS3 downloads data from S3 and verifies its integrity.
func RestoreFromS3(ctx context.Context, downloader S3Downloader, config RestoreConfig) error {
	// Ensure destination directory exists
	destDir := filepath.Dir(config.DestPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", destDir, err)
	}

	// Download from S3
	if err := downloader.Download(ctx, config.Bucket, config.Key, config.DestPath); err != nil {
		return fmt.Errorf("failed to download from S3: %w", err)
	}

	// Verify integrity
	if err := VerifyRestoredData(config.DestPath, config.ExpectedChecksum); err != nil {
		// Clean up corrupted file
		os.Remove(config.DestPath)
		return fmt.Errorf("downloaded data integrity check failed: %w", err)
	}

	return nil
}

// VerifyRestoredData checks if the restored file matches the expected checksum.
func VerifyRestoredData(path, expectedChecksum string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}
	actualChecksum := fmt.Sprintf("%x", hasher.Sum(nil))

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}
