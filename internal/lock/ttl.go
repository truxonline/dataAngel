package lock

import (
	"context"
	"fmt"
	"time"
)

// TTLLock représente un lock avec TTL (Time To Live)
type TTLLock struct {
	*S3LockMock
	ttl         time.Duration
	lastRenewal time.Time
}

// NewTTLLock crée un nouveau lock avec TTL
func NewTTLLock(bucket, key string, ttl time.Duration) *TTLLock {
	return &TTLLock{
		S3LockMock:  NewS3LockMock(bucket, key),
		ttl:         ttl,
		lastRenewal: time.Now(),
	}
}

// Acquire tente d'acquérir le lock avec TTL
func (l *TTLLock) Acquire(ctx context.Context, timeout time.Duration) error {
	err := l.S3LockMock.Acquire(ctx, timeout)
	if err != nil {
		return err
	}

	l.lastRenewal = time.Now()
	return nil
}

// Renew renouvelle le lock (reset TTL)
func (l *TTLLock) Renew(ctx context.Context) error {
	if !l.S3LockMock.isLocked {
		return fmt.Errorf("lock n'est pas acquis")
	}

	l.lastRenewal = time.Now()
	return nil
}

// IsExpired vérifie si le lock est expiré
func (l *TTLLock) IsExpired() bool {
	if !l.S3LockMock.isLocked {
		return true
	}

	return time.Since(l.lastRenewal) > l.ttl
}

// Steal tente de voler un lock expiré
func (l *TTLLock) Steal(ctx context.Context) error {
	if !l.IsExpired() {
		return fmt.Errorf("lock n'est pas expiré")
	}

	if l.S3LockMock.isLocked {
		l.S3LockMock.isLocked = false
		l.S3LockMock.metadata = make(map[string]string)
	}

	return l.Acquire(ctx, 0)
}

// GetMetadata récupère les métadonnées du lock avec TTL
func (l *TTLLock) GetMetadata(ctx context.Context) (map[string]string, error) {
	metadata, err := l.S3LockMock.GetMetadata(ctx)
	if err != nil {
		return nil, err
	}

	metadata["ttl"] = l.ttl.String()
	metadata["last-renewal"] = l.lastRenewal.Format(time.RFC3339)
	metadata["expires-at"] = l.lastRenewal.Add(l.ttl).Format(time.RFC3339)

	return metadata, nil
}
