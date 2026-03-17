package lock

import (
	"context"
	"fmt"
	"time"
)

// LockInterface définit l'interface pour les locks distribués
type LockInterface interface {
	Acquire(ctx context.Context, timeout time.Duration) error
	Release(ctx context.Context) error
	IsLocked(ctx context.Context) (bool, error)
	GetMetadata(ctx context.Context) (map[string]string, error)
}

// S3LockMock est une implémentation mock de LockInterface pour les tests
type S3LockMock struct {
	bucket    string
	key       string
	leaseID   string
	createdAt time.Time
	isLocked  bool
	metadata  map[string]string
}

// NewS3LockMock crée un nouveau lock mock
func NewS3LockMock(bucket, key string) *S3LockMock {
	return &S3LockMock{
		bucket:    bucket,
		key:       key,
		leaseID:   fmt.Sprintf("lock-%d", time.Now().UnixNano()),
		createdAt: time.Now(),
		isLocked:  false,
		metadata:  make(map[string]string),
	}
}

// Acquire tente d'acquérir le lock
func (l *S3LockMock) Acquire(ctx context.Context, timeout time.Duration) error {
	if l.isLocked {
		return fmt.Errorf("lock déjà acquis")
	}

	l.isLocked = true
	l.metadata["lease-id"] = l.leaseID
	l.metadata["created-at"] = l.createdAt.Format(time.RFC3339)

	return nil
}

// Release libère le lock
func (l *S3LockMock) Release(ctx context.Context) error {
	if !l.isLocked {
		return fmt.Errorf("lock n'est pas acquis")
	}

	l.isLocked = false
	l.metadata = make(map[string]string)

	return nil
}

// IsLocked vérifie si le lock est actif
func (l *S3LockMock) IsLocked(ctx context.Context) (bool, error) {
	return l.isLocked, nil
}

// GetMetadata récupère les métadonnées du lock
func (l *S3LockMock) GetMetadata(ctx context.Context) (map[string]string, error) {
	if !l.isLocked {
		return nil, fmt.Errorf("lock n'est pas acquis")
	}

	return l.metadata, nil
}

// S3Lock implémentation réelle (placeholder pour l'intégration S3)
type S3Lock struct {
	bucket    string
	key       string
	leaseID   string
	createdAt time.Time
}

// NewS3Lock crée un nouveau lock S3
func NewS3Lock(bucket, key string) (*S3Lock, error) {
	return &S3Lock{
		bucket:    bucket,
		key:       key,
		leaseID:   fmt.Sprintf("lock-%d", time.Now().UnixNano()),
		createdAt: time.Now(),
	}, nil
}

// Acquire tente d'acquérir le lock (placeholder - implémentation S3 à ajouter)
func (l *S3Lock) Acquire(ctx context.Context, timeout time.Duration) error {
	return fmt.Errorf("implémentation S3 non disponible - utiliser S3LockMock pour les tests")
}

// Release libère le lock (placeholder - implémentation S3 à ajouter)
func (l *S3Lock) Release(ctx context.Context) error {
	return fmt.Errorf("implémentation S3 non disponible - utiliser S3LockMock pour les tests")
}

// IsLocked vérifie si le lock est actif (placeholder - implémentation S3 à ajouter)
func (l *S3Lock) IsLocked(ctx context.Context) (bool, error) {
	return false, fmt.Errorf("implémentation S3 non disponible - utiliser S3LockMock pour les tests")
}

// GetMetadata récupère les métadonnées du lock (placeholder - implémentation S3 à ajouter)
func (l *S3Lock) GetMetadata(ctx context.Context) (map[string]string, error) {
	return nil, fmt.Errorf("implémentation S3 non disponible - utiliser S3LockMock pour les tests")
}
