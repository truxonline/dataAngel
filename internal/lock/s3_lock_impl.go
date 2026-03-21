package lock

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type LockMetadata struct {
	PodName    string    `json:"pod_name"`
	PodUID     string    `json:"pod_uid"`
	Hostname   string    `json:"hostname"`
	AcquiredAt time.Time `json:"acquired_at"`
	TTL        int       `json:"ttl_seconds"`
}

type S3LockConfig struct {
	Bucket   string
	Key      string
	Endpoint string
	TTL      time.Duration
}

type S3LockReal struct {
	client   *s3.Client
	bucket   string
	key      string
	ttl      time.Duration
	metadata LockMetadata
	acquired bool
}

func NewS3LockReal(ctx context.Context, cfg S3LockConfig) (*S3LockReal, error) {
	// Default region to us-east-1 for custom endpoints (MinIO, etc.)
	// AWS SDK v2 requires a region even for non-AWS S3-compatible storage
	var configOpts []func(*config.LoadOptions) error
	if cfg.Endpoint != "" && os.Getenv("AWS_REGION") == "" {
		configOpts = append(configOpts, config.WithRegion("us-east-1"))
	}

	awsCfg, err := config.LoadDefaultConfig(ctx, configOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	var s3Client *s3.Client
	if cfg.Endpoint != "" {
		s3Client = s3.NewFromConfig(awsCfg, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.Endpoint)
			o.UsePathStyle = true
		})
	} else {
		s3Client = s3.NewFromConfig(awsCfg)
	}

	hostname, _ := os.Hostname()
	podName := os.Getenv("POD_NAME")
	if podName == "" {
		podName = hostname
	}
	podUID := os.Getenv("POD_UID")

	return &S3LockReal{
		client: s3Client,
		bucket: cfg.Bucket,
		key:    cfg.Key,
		ttl:    cfg.TTL,
		metadata: LockMetadata{
			PodName:  podName,
			PodUID:   podUID,
			Hostname: hostname,
			TTL:      int(cfg.TTL.Seconds()),
		},
	}, nil
}

func (l *S3LockReal) Acquire(ctx context.Context, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	baseInterval := 2 * time.Second

	for time.Now().Before(deadline) {
		locked, err := l.tryAcquire(ctx)
		if err != nil {
			return fmt.Errorf("failed to try acquire lock: %w", err)
		}

		if locked {
			l.acquired = true
			l.metadata.AcquiredAt = time.Now()
			return nil
		}

		expired, err := l.isExpired(ctx)
		if err != nil {
			return fmt.Errorf("failed to check lock expiration: %w", err)
		}

		if expired {
			if err := l.forceRelease(ctx); err != nil {
				return fmt.Errorf("failed to release expired lock: %w", err)
			}
			continue
		}

		// Jitter to avoid thundering herd on cluster-wide restarts (#34)
		jitter := time.Duration(rand.Int63n(int64(baseInterval)))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(baseInterval + jitter):
		}
	}

	return fmt.Errorf("timeout acquiring lock after %v", timeout)
}

func (l *S3LockReal) tryAcquire(ctx context.Context) (bool, error) {
	l.metadata.AcquiredAt = time.Now()
	body, err := json.Marshal(l.metadata)
	if err != nil {
		return false, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = l.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:            aws.String(l.bucket),
		Key:               aws.String(l.key),
		Body:              bytes.NewReader(body),
		ContentType:       aws.String("application/json"),
		IfNoneMatch:       aws.String("*"),
		ChecksumAlgorithm: types.ChecksumAlgorithmSha256,
	})

	if err != nil {
		if isConditionFailed(err) {
			return false, nil
		}
		return false, fmt.Errorf("S3 PutObject failed: %w", err)
	}

	return true, nil
}

func (l *S3LockReal) Renew(ctx context.Context) error {
	if !l.acquired {
		return fmt.Errorf("lock not acquired, cannot renew")
	}

	l.metadata.AcquiredAt = time.Now()
	body, err := json.Marshal(l.metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	_, err = l.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(l.bucket),
		Key:         aws.String(l.key),
		Body:        bytes.NewReader(body),
		ContentType: aws.String("application/json"),
	})

	if err != nil {
		return fmt.Errorf("failed to renew lock: %w", err)
	}

	return nil
}

func (l *S3LockReal) Release(ctx context.Context) error {
	if !l.acquired {
		return fmt.Errorf("lock not acquired, cannot release")
	}

	_, err := l.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(l.bucket),
		Key:    aws.String(l.key),
	})

	if err != nil {
		return fmt.Errorf("failed to delete lock object: %w", err)
	}

	l.acquired = false
	return nil
}

func (l *S3LockReal) forceRelease(ctx context.Context) error {
	_, err := l.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(l.bucket),
		Key:    aws.String(l.key),
	})
	return err
}

func (l *S3LockReal) IsLocked(ctx context.Context) (bool, error) {
	_, err := l.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(l.bucket),
		Key:    aws.String(l.key),
	})

	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) || isNoSuchKey(err) {
			return false, nil
		}
		return false, fmt.Errorf("failed to check lock: %w", err)
	}

	return true, nil
}

func (l *S3LockReal) GetMetadata(ctx context.Context) (map[string]string, error) {
	result, err := l.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(l.bucket),
		Key:    aws.String(l.key),
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get lock object: %w", err)
	}
	defer result.Body.Close()

	body, err := io.ReadAll(result.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock body: %w", err)
	}

	var meta LockMetadata
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	return map[string]string{
		"pod_name":    meta.PodName,
		"pod_uid":     meta.PodUID,
		"hostname":    meta.Hostname,
		"acquired_at": meta.AcquiredAt.Format(time.RFC3339),
		"ttl":         fmt.Sprintf("%d", meta.TTL),
	}, nil
}

func (l *S3LockReal) isExpired(ctx context.Context) (bool, error) {
	meta, err := l.GetMetadata(ctx)
	if err != nil {
		var notFound *types.NotFound
		if errors.As(err, &notFound) || isNoSuchKey(err) {
			return false, nil
		}
		return false, err
	}

	acquiredAt, err := time.Parse(time.RFC3339, meta["acquired_at"])
	if err != nil {
		return false, fmt.Errorf("failed to parse acquired_at: %w", err)
	}

	ttl, _ := time.ParseDuration(meta["ttl"] + "s")
	if ttl == 0 {
		ttl = l.ttl
	}

	expiration := acquiredAt.Add(ttl)
	return time.Now().After(expiration), nil
}

func isConditionFailed(err error) bool {
	if err == nil {
		return false
	}
	errMsg := err.Error()
	return containsString(errMsg, "PreconditionFailed") ||
		containsString(errMsg, "IfNoneMatch") ||
		containsString(errMsg, "412") ||
		containsString(errMsg, "At least one of the pre-conditions you specified did not hold")
}

func isNoSuchKey(err error) bool {
	var noSuchKey *types.NoSuchKey
	return errors.As(err, &noSuchKey) ||
		(err != nil && containsString(err.Error(), "NoSuchKey"))
}

func containsString(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr || (len(s) >= len(substr) && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
