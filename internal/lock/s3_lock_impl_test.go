package lock

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestS3LockReal_RegionDefault(t *testing.T) {
	originalRegion := os.Getenv("AWS_REGION")
	defer func() {
		if originalRegion != "" {
			os.Setenv("AWS_REGION", originalRegion)
		} else {
			os.Unsetenv("AWS_REGION")
		}
	}()

	os.Unsetenv("AWS_REGION")

	ctx := context.Background()
	cfg := S3LockConfig{
		Bucket:   "test-bucket",
		Key:      ".locks/test",
		Endpoint: "http://minio.local:9000",
		TTL:      60 * time.Second,
	}

	_, err := NewS3LockReal(ctx, cfg)

	if err != nil && containsString(err.Error(), "region must be set") {
		t.Errorf("AWS_REGION should be defaulted for custom endpoint, got: %v", err)
	}
}

func TestS3LockReal_AcquireRelease(t *testing.T) {
	t.Skip("Integration test: requires MinIO or S3 endpoint")

	ctx := context.Background()
	cfg := S3LockConfig{
		Bucket:   "test-bucket",
		Key:      ".locks/test-deployment",
		Endpoint: "http://localhost:9000",
		TTL:      60 * time.Second,
	}

	lock, err := NewS3LockReal(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create lock: %v", err)
	}

	if err := lock.Acquire(ctx, 10*time.Second); err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}

	if !lock.acquired {
		t.Error("Lock should be marked as acquired")
	}

	locked, err := lock.IsLocked(ctx)
	if err != nil {
		t.Errorf("Failed to check lock status: %v", err)
	}
	if !locked {
		t.Error("Lock should exist in S3")
	}

	if err := lock.Release(ctx); err != nil {
		t.Errorf("Failed to release lock: %v", err)
	}

	if lock.acquired {
		t.Error("Lock should be marked as released")
	}

	locked, err = lock.IsLocked(ctx)
	if err != nil {
		t.Errorf("Failed to check lock status: %v", err)
	}
	if locked {
		t.Error("Lock should not exist in S3 after release")
	}
}

func TestS3LockReal_Conflict(t *testing.T) {
	t.Skip("Integration test: requires MinIO or S3 endpoint")

	ctx := context.Background()
	cfg := S3LockConfig{
		Bucket:   "test-bucket",
		Key:      ".locks/test-conflict",
		Endpoint: "http://localhost:9000",
		TTL:      60 * time.Second,
	}

	lock1, err := NewS3LockReal(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create lock1: %v", err)
	}

	lock2, err := NewS3LockReal(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create lock2: %v", err)
	}

	if err := lock1.Acquire(ctx, 5*time.Second); err != nil {
		t.Fatalf("Lock1 failed to acquire: %v", err)
	}
	defer lock1.Release(ctx)

	if err := lock2.Acquire(ctx, 2*time.Second); err == nil {
		t.Error("Lock2 should fail to acquire (lock1 holds it)")
	}
}

func TestS3LockReal_Expiration(t *testing.T) {
	t.Skip("Integration test: requires MinIO or S3 endpoint")

	ctx := context.Background()
	cfg := S3LockConfig{
		Bucket:   "test-bucket",
		Key:      ".locks/test-expiration",
		Endpoint: "http://localhost:9000",
		TTL:      2 * time.Second,
	}

	lock1, err := NewS3LockReal(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create lock1: %v", err)
	}

	if err := lock1.Acquire(ctx, 5*time.Second); err != nil {
		t.Fatalf("Lock1 failed to acquire: %v", err)
	}

	time.Sleep(3 * time.Second)

	lock2, err := NewS3LockReal(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create lock2: %v", err)
	}

	if err := lock2.Acquire(ctx, 5*time.Second); err != nil {
		t.Fatalf("Lock2 should acquire after expiration: %v", err)
	}
	defer lock2.Release(ctx)
}

func TestS3LockReal_Renew(t *testing.T) {
	t.Skip("Integration test: requires MinIO or S3 endpoint")

	ctx := context.Background()
	cfg := S3LockConfig{
		Bucket:   "test-bucket",
		Key:      ".locks/test-renew",
		Endpoint: "http://localhost:9000",
		TTL:      5 * time.Second,
	}

	lock, err := NewS3LockReal(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create lock: %v", err)
	}

	if err := lock.Acquire(ctx, 5*time.Second); err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	defer lock.Release(ctx)

	time.Sleep(3 * time.Second)

	if err := lock.Renew(ctx); err != nil {
		t.Errorf("Failed to renew lock: %v", err)
	}

	meta, err := lock.GetMetadata(ctx)
	if err != nil {
		t.Errorf("Failed to get metadata: %v", err)
	}

	acquiredAt, err := time.Parse(time.RFC3339, meta["acquired_at"])
	if err != nil {
		t.Errorf("Failed to parse acquired_at: %v", err)
	}

	if time.Since(acquiredAt) > 1*time.Second {
		t.Errorf("Lock should have been renewed recently, got age: %v", time.Since(acquiredAt))
	}
}

func TestS3LockReal_Metadata(t *testing.T) {
	t.Skip("Integration test: requires MinIO or S3 endpoint")

	ctx := context.Background()
	cfg := S3LockConfig{
		Bucket:   "test-bucket",
		Key:      ".locks/test-metadata",
		Endpoint: "http://localhost:9000",
		TTL:      60 * time.Second,
	}

	lock, err := NewS3LockReal(ctx, cfg)
	if err != nil {
		t.Fatalf("Failed to create lock: %v", err)
	}

	if err := lock.Acquire(ctx, 5*time.Second); err != nil {
		t.Fatalf("Failed to acquire lock: %v", err)
	}
	defer lock.Release(ctx)

	meta, err := lock.GetMetadata(ctx)
	if err != nil {
		t.Errorf("Failed to get metadata: %v", err)
	}

	required := []string{"pod_name", "hostname", "acquired_at", "ttl"}
	for _, key := range required {
		if _, ok := meta[key]; !ok {
			t.Errorf("Metadata missing key: %s", key)
		}
	}
}
