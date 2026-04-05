package s3

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/brycesharrits/fam-cal-insta/internal/config"
)

type Storage struct {
	client   *s3.Client
	presign  *s3.PresignClient
	bucket   string
	endpoint string
}

func New(ctx context.Context, cfg *config.Config) (*Storage, error) {
	opts := []func(*awsconfig.LoadOptions) error{
		awsconfig.WithRegion(cfg.S3Region),
		awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		),
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("loading aws config: %w", err)
	}

	clientOpts := []func(*s3.Options){}
	if cfg.S3Endpoint != "" {
		// MinIO or other S3-compatible endpoints
		clientOpts = append(clientOpts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(cfg.S3Endpoint)
			o.UsePathStyle = true // Required for MinIO
		})
	}

	client := s3.NewFromConfig(awsCfg, clientOpts...)

	return &Storage{
		client:   client,
		presign:  s3.NewPresignClient(client),
		bucket:   cfg.S3Bucket,
		endpoint: cfg.S3Endpoint,
	}, nil
}

func (s *Storage) PutObject(ctx context.Context, key string, data io.Reader, contentType string) (string, error) {
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s.bucket),
		Key:         aws.String(key),
		Body:        data,
		ContentType: aws.String(contentType),
		ACL:         types.ObjectCannedACLPublicRead,
	})
	if err != nil {
		return "", fmt.Errorf("putting object %s: %w", key, err)
	}
	return s.PublicURL(key), nil
}

func (s *Storage) GetPresignedUploadURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := s.presign.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presigning upload URL for %s: %w", key, err)
	}
	return req.URL, nil
}

func (s *Storage) GetPresignedDownloadURL(ctx context.Context, key string, ttl time.Duration) (string, error) {
	req, err := s.presign.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	}, s3.WithPresignExpires(ttl))
	if err != nil {
		return "", fmt.Errorf("presigning download URL for %s: %w", key, err)
	}
	return req.URL, nil
}

func (s *Storage) DeleteObject(ctx context.Context, key string) error {
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(key),
	})
	return err
}

func (s *Storage) PublicURL(key string) string {
	if s.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", s.endpoint, s.bucket, key)
	}
	return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.bucket, key)
}
