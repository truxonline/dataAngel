package lock

import (
	"context"
	"testing"
	"time"
)

func TestTTLLockAcquire(t *testing.T) {
	lock := NewTTLLock("test-bucket", "test-key", 5*time.Second)
	ctx := context.Background()

	err := lock.Acquire(ctx, 5*time.Second)
	if err != nil {
		t.Errorf("L'acquisition du lock avec TTL a échoué: %v", err)
	}

	if !lock.S3LockMock.isLocked {
		t.Errorf("Le lock devrait être acquis")
	}
}

func TestTTLLockRenew(t *testing.T) {
	lock := NewTTLLock("test-bucket", "test-key", 5*time.Second)
	ctx := context.Background()

	err := lock.Acquire(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("L'acquisition du lock a échoué: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	err = lock.Renew(ctx)
	if err != nil {
		t.Errorf("Le renouvellement du lock a échoué: %v", err)
	}

	if lock.lastRenewal.IsZero() {
		t.Errorf("La date de renouvellement devrait être mise à jour")
	}
}

func TestTTLLockExpired(t *testing.T) {
	lock := NewTTLLock("test-bucket", "test-key", 100*time.Millisecond)
	ctx := context.Background()

	err := lock.Acquire(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("L'acquisition du lock a échoué: %v", err)
	}

	if lock.IsExpired() {
		t.Errorf("Le lock ne devrait pas être expiré immédiatement")
	}

	time.Sleep(200 * time.Millisecond)

	if !lock.IsExpired() {
		t.Errorf("Le lock devrait être expiré après le TTL")
	}
}

func TestTTLLockSteal(t *testing.T) {
	lock := NewTTLLock("test-bucket", "test-key", 100*time.Millisecond)
	ctx := context.Background()

	err := lock.Acquire(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("L'acquisition du lock a échoué: %v", err)
	}

	time.Sleep(200 * time.Millisecond)

	err = lock.Steal(ctx)
	if err != nil {
		t.Errorf("Le vol du lock expiré a échoué: %v", err)
	}

	if !lock.S3LockMock.isLocked {
		t.Errorf("Le lock devrait être acquis après le vol")
	}
}

func TestTTLLockMetadata(t *testing.T) {
	lock := NewTTLLock("test-bucket", "test-key", 5*time.Second)
	ctx := context.Background()

	err := lock.Acquire(ctx, 5*time.Second)
	if err != nil {
		t.Fatalf("L'acquisition du lock a échoué: %v", err)
	}

	metadata, err := lock.GetMetadata(ctx)
	if err != nil {
		t.Errorf("La récupération des métadonnées a échoué: %v", err)
	}

	if _, ok := metadata["ttl"]; !ok {
		t.Errorf("Les métadonnées devraient contenir ttl")
	}

	if _, ok := metadata["last-renewal"]; !ok {
		t.Errorf("Les métadonnées devraient contenir last-renewal")
	}

	if _, ok := metadata["expires-at"]; !ok {
		t.Errorf("Les métadonnées devraient contenir expires-at")
	}
}
