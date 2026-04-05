package storage

import (
	"context"
	"io"
	"time"
)

// ObjectStorage abstracts S3-compatible object storage.
// Works with AWS S3, Cloudflare R2, MinIO, etc.
type ObjectStorage interface {
	// PutObject uploads data and returns the public URL.
	PutObject(ctx context.Context, key string, data io.Reader, contentType string) (url string, err error)

	// GetPresignedUploadURL returns a time-limited URL clients can PUT directly to.
	GetPresignedUploadURL(ctx context.Context, key string, ttl time.Duration) (string, error)

	// GetPresignedDownloadURL returns a time-limited URL for downloading a private object.
	GetPresignedDownloadURL(ctx context.Context, key string, ttl time.Duration) (string, error)

	// DeleteObject removes an object.
	DeleteObject(ctx context.Context, key string) error

	// PublicURL returns the permanent public URL for an object (for public buckets).
	PublicURL(key string) string
}
