package lock

import (
	"context"
	"testing"
	"time"
)

func TestS3LockAcquire(t *testing.T) {
	lock := NewS3LockMock("test-bucket", "test-key")
	ctx := context.Background()

	err := lock.Acquire(ctx, 5*time.Second)
	if err != nil {
		t.Errorf("L'acquisition du lock a échoué: %v", err)
	}

	if !lock.isLocked {
		t.Errorf("Le lock devrait être acquis")
	}
}

func TestS3LockRelease(t *testing.T) {
	lock := NewS3LockMock("test-bucket", "test-key")
	ctx := context.Background()

	err := lock.Acquire(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("L'acquisition du lock a échoué: %v", err)
	}

	err = lock.Release(ctx)
	if err != nil {
		t.Errorf("La libération du lock a échoué: %v", err)
	}

	if lock.isLocked {
		t.Errorf("Le lock devrait être libéré")
	}
}

func TestS3LockContention(t *testing.T) {
	lock := NewS3LockMock("test-bucket", "test-key")
	ctx := context.Background()

	err := lock.Acquire(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("L'acquisition du lock a échoué: %v", err)
	}

	err = lock.Acquire(ctx, 5*time.Second)
	if err == nil {
		t.Errorf("L'acquisition d'un lock déjà acquis devrait échouer")
	}
}

func TestS3LockMetadata(t *testing.T) {
	lock := NewS3LockMock("test-bucket", "test-key")
	ctx := context.Background()

	err := lock.Acquire(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("L'acquisition du lock a échoué: %v", err)
	}

	metadata, err := lock.GetMetadata(ctx)
	if err != nil {
		t.Errorf("La récupération des métadonnées a échoué: %v", err)
	}

	if _, ok := metadata["lease-id"]; !ok {
		t.Errorf("Les métadonnées devraient contenir lease-id")
	}

	if _, ok := metadata["created-at"]; !ok {
		t.Errorf("Les métadonnées devraient contenir created-at")
	}
}
