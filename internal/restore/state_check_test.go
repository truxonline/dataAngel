package restore

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckDataHealth(t *testing.T) {
	t.Run("should detect healthy data", func(t *testing.T) {
		// Arrange
		state := DataState{
			Exists:    true,
			Checksum:  "valid-checksum",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Size:      1024,
			Path:      "/path/to/data.db",
		}

		// Act
		healthy, err := CheckDataHealth(state)

		// Assert
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !healthy {
			t.Error("Expected data to be healthy")
		}
	})

	t.Run("should detect unhealthy data (missing)", func(t *testing.T) {
		// Arrange
		state := DataState{
			Exists: false,
		}

		// Act
		healthy, err := CheckDataHealth(state)

		// Assert
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if healthy {
			t.Error("Expected data to be unhealthy (missing)")
		}
	})

	t.Run("should detect unhealthy data (corrupted checksum)", func(t *testing.T) {
		// Arrange
		state := DataState{
			Exists:    true,
			Checksum:  "", // Empty checksum represents corruption or missing validation
			Timestamp: time.Now(),
			Size:      1024,
			Path:      "/path/to/data.db",
		}

		// Act
		healthy, err := CheckDataHealth(state)

		// Assert
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if healthy {
			t.Error("Expected data to be unhealthy (corrupted checksum)")
		}
	})
}

func TestCompareStates(t *testing.T) {
	t.Run("should skip restore when local is newer than remote", func(t *testing.T) {
		local := DataState{
			Exists:    true,
			Checksum:  "abc123",
			Timestamp: time.Now(),
			Size:      1024,
			Path:      "/local/data.db",
		}
		remote := DataState{
			Exists:    true,
			Checksum:  "abc123",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Size:      1024,
			Path:      "/remote/data.db",
		}

		decision := CompareStates(local, remote)

		if decision != DecisionSkip {
			t.Errorf("Expected DecisionSkip, got %v", decision)
		}
	})

	t.Run("should need restore when remote is newer", func(t *testing.T) {
		local := DataState{
			Exists:    true,
			Checksum:  "abc123",
			Timestamp: time.Now().Add(-1 * time.Hour),
			Size:      1024,
			Path:      "/local/data.db",
		}
		remote := DataState{
			Exists:    true,
			Checksum:  "abc123",
			Timestamp: time.Now(),
			Size:      1024,
			Path:      "/remote/data.db",
		}

		decision := CompareStates(local, remote)

		if decision != DecisionRestore {
			t.Errorf("Expected DecisionRestore, got %v", decision)
		}
	})

	t.Run("should need restore when local is missing", func(t *testing.T) {
		local := DataState{Exists: false}
		remote := DataState{
			Exists:    true,
			Checksum:  "abc123",
			Timestamp: time.Now(),
			Size:      1024,
			Path:      "/remote/data.db",
		}

		decision := CompareStates(local, remote)

		if decision != DecisionRestore {
			t.Errorf("Expected DecisionRestore, got %v", decision)
		}
	})

	t.Run("should skip restore when both states are equal", func(t *testing.T) {
		fixedTime := time.Now().Truncate(time.Second)
		local := DataState{
			Exists:    true,
			Checksum:  "abc123",
			Timestamp: fixedTime,
			Size:      1024,
			Path:      "/local/data.db",
		}
		remote := DataState{
			Exists:    true,
			Checksum:  "abc123",
			Timestamp: fixedTime,
			Size:      1024,
			Path:      "/remote/data.db",
		}

		decision := CompareStates(local, remote)

		if decision != DecisionSkip {
			t.Errorf("Expected DecisionSkip, got %v", decision)
		}
	})

	t.Run("should need restore when local is corrupted", func(t *testing.T) {
		local := DataState{
			Exists:    true,
			Checksum:  "", // Empty checksum means corrupted
			Timestamp: time.Now(),
			Size:      1024,
			Path:      "/local/data.db",
		}
		remote := DataState{
			Exists:    true,
			Checksum:  "abc123",
			Timestamp: time.Now(),
			Size:      1024,
			Path:      "/remote/data.db",
		}

		decision := CompareStates(local, remote)

		if decision != DecisionCorrupted {
			t.Errorf("Expected DecisionCorrupted, got %v", decision)
		}
	})
}

func TestGetLocalState(t *testing.T) {
	t.Run("should read valid file and compute checksum", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "data.db")
		content := []byte("test data content")
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
		expectedChecksum := fmt.Sprintf("%x", sha256.Sum256(content))

		// Act
		state, err := GetLocalState(path)

		// Assert
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !state.Exists {
			t.Error("Expected data to exist")
		}
		if state.Checksum != expectedChecksum {
			t.Errorf("Expected checksum %s, got %s", expectedChecksum, state.Checksum)
		}
		if state.Size != int64(len(content)) {
			t.Errorf("Expected size %d, got %d", len(content), state.Size)
		}
		if state.Path != path {
			t.Errorf("Expected path %s, got %s", path, state.Path)
		}
	})

	t.Run("should return Exists=false for missing path", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "missing.db")

		// Act
		state, err := GetLocalState(path)

		// Assert
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if state.Exists {
			t.Error("Expected data to not exist")
		}
	})

	t.Run("should handle empty file", func(t *testing.T) {
		// Arrange
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "empty.db")
		if err := os.WriteFile(path, []byte{}, 0644); err != nil {
			t.Fatal(err)
		}
		expectedChecksum := fmt.Sprintf("%x", sha256.Sum256([]byte{}))

		// Act
		state, err := GetLocalState(path)

		// Assert
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !state.Exists {
			t.Error("Expected data to exist (even if empty)")
		}
		if state.Checksum != expectedChecksum {
			t.Errorf("Expected checksum %s, got %s", expectedChecksum, state.Checksum)
		}
		if state.Size != 0 {
			t.Errorf("Expected size 0, got %d", state.Size)
		}
	})
}
