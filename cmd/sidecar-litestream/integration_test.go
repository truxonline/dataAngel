package main

import (
	"os"
	"testing"
)

func TestStreamSQLiteToS3_Success(t *testing.T) {
	// Arrange - créer une base SQLite temporaire
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	// Créer une base SQLite vide
	file, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	file.Close()

	s3URI := "s3://test-bucket/backups/test.db"

	// Act
	err = StreamSQLiteToS3(dbPath, s3URI)

	// Assert
	if err != nil {
		t.Errorf("Expected successful streaming, got error: %v", err)
	}
}

func TestStreamSQLiteToS3_MissingDB(t *testing.T) {
	// Arrange
	dbPath := "/tmp/missing.db"
	s3URI := "s3://test-bucket/backups/test.db"

	// Act
	err := StreamSQLiteToS3(dbPath, s3URI)

	// Assert
	if err == nil {
		t.Error("Expected error for missing database")
	}
}

func TestRestoreFromS3_Success(t *testing.T) {
	// Arrange
	s3URI := "s3://test-bucket/backups/test.db"
	restorePath := t.TempDir() + "/restore.db"

	// Act
	err := RestoreFromS3(s3URI, restorePath)

	// Assert
	if err != nil {
		t.Errorf("Expected successful restore, got error: %v", err)
	}
}
