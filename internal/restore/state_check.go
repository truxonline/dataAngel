package restore

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"time"
)

// DataState represents the state of data.
type DataState struct {
	Exists    bool
	Checksum  string
	Timestamp time.Time
	Size      int64
	Path      string
}

// RestoreDecision represents the decision made after comparing local and remote states.
type RestoreDecision int

const (
	DecisionSkip RestoreDecision = iota
	DecisionRestore
	DecisionCorrupted
)

// CheckDataHealth vérifie si les données sont saines.
// Retourne (true, nil) si saines, (false, nil) si non saines, (false, error) si erreur.
func CheckDataHealth(state DataState) (bool, error) {
	if !state.Exists {
		return false, nil
	}

	if state.Checksum == "" {
		return false, nil
	}

	return true, nil
}

// S3StateClient defines the interface for fetching remote state from S3.
type S3StateClient interface {
	GetRemoteState(ctx context.Context, bucket, path string) (DataState, error)
}

// MockS3StateClient is a mock implementation of S3StateClient for testing.
type MockS3StateClient struct {
	remoteState DataState
	err         error
}

// GetRemoteState returns the mock remote state or error.
func (m *MockS3StateClient) GetRemoteState(ctx context.Context, bucket, path string) (DataState, error) {
	return m.remoteState, m.err
}

// CompareStates compares local and remote data states and returns a restore decision.
func CompareStates(local, remote DataState) RestoreDecision {
	if !local.Exists {
		return DecisionRestore
	}

	if local.Checksum == "" {
		return DecisionCorrupted
	}

	if local.Checksum != remote.Checksum {
		return DecisionRestore
	}

	if local.Timestamp.Before(remote.Timestamp) {
		return DecisionRestore
	}

	return DecisionSkip
}

// GetLocalState reads the local file and returns its state.
func GetLocalState(path string) (DataState, error) {
	state := DataState{Path: path}

	// Check if file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		state.Exists = false
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("failed to stat file: %w", err)
	}

	// File exists
	state.Exists = true
	state.Size = info.Size()
	state.Timestamp = info.ModTime()

	// Compute checksum
	file, err := os.Open(path)
	if err != nil {
		return state, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return state, fmt.Errorf("failed to compute checksum: %w", err)
	}
	state.Checksum = fmt.Sprintf("%x", hasher.Sum(nil))

	return state, nil
}
