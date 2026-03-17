package s3

import "time"

// S3Config holds S3 connection configuration
type S3Config struct {
	Bucket   string
	Region   string
	Endpoint string
}

// BackupInfo represents metadata about a backup stored in S3
type BackupInfo struct {
	Name         string
	Size         int64
	LastModified time.Time
	Checksum     string
	Path         string
}

// BackupStatus represents the status of backups
type BackupStatus struct {
	Backups []BackupInfo
	Count   int
	Healthy bool
}
