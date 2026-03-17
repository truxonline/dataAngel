package main

import (
	"os"
	"testing"
)

func TestStreamSQLiteToS3_Success(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := tmpDir + "/test.db"

	file, err := os.Create(dbPath)
	if err != nil {
		t.Fatalf("Failed to create temp db: %v", err)
	}
	file.Close()

	t.Setenv("DATA_GUARD_ENABLED", "true")
	t.Setenv("DATA_GUARD_BUCKET", "test-bucket")
	t.Setenv("DATA_GUARD_SQLITE_PATHS", dbPath)

	err = StreamSQLiteToS3()

	if err != nil {
		t.Errorf("Expected successful streaming, got error: %v", err)
	}
}

func TestStreamSQLiteToS3_MissingDB(t *testing.T) {
	t.Setenv("DATA_GUARD_ENABLED", "true")
	t.Setenv("DATA_GUARD_BUCKET", "test-bucket")
	t.Setenv("DATA_GUARD_SQLITE_PATHS", "/tmp/missing.db")

	err := StreamSQLiteToS3()

	if err != nil {
		t.Errorf("Expected no error for missing database (should skip), got: %v", err)
	}
}

func TestRestoreFromS3_Success(t *testing.T) {
	t.Setenv("DATA_GUARD_ENABLED", "true")
	t.Setenv("DATA_GUARD_BUCKET", "test-bucket")
	t.Setenv("DATA_GUARD_SQLITE_PATHS", "/tmp/test.db")

	err := RestoreFromS3()

	if err != nil {
		t.Errorf("Expected successful restore, got error: %v", err)
	}
}
