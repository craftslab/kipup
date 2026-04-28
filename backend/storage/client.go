package storage

import (
	"context"
	"io"
	"net/url"
	"time"

	"github.com/craftslab/s3c/backend/config"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Client wraps the MinIO client and exposes S3 operations.
type Client struct {
	mc *minio.Client
}

// NewClient creates a new S3/MinIO client from configuration.
func NewClient(cfg *config.Config) (*Client, error) {
	mc, err := minio.New(cfg.S3Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.S3AccessKey, cfg.S3SecretKey, ""),
		Secure: cfg.S3UseSSL,
		Region: cfg.S3Region,
	})
	if err != nil {
		return nil, err
	}
	return &Client{mc: mc}, nil
}

// ListBuckets returns metadata for all buckets.
func (c *Client) ListBuckets(ctx context.Context) ([]minio.BucketInfo, error) {
	return c.mc.ListBuckets(ctx)
}

// MakeBucket creates a new bucket in the given region.
func (c *Client) MakeBucket(ctx context.Context, bucket, region string) error {
	return c.mc.MakeBucket(ctx, bucket, minio.MakeBucketOptions{Region: region})
}

// RemoveBucket deletes an empty bucket.
func (c *Client) RemoveBucket(ctx context.Context, bucket string) error {
	return c.mc.RemoveBucket(ctx, bucket)
}

// ListObjects lists objects (and common-prefix "directories") under a prefix.
// It uses delimiter "/" so sub-folder entries are returned as directory stubs.
func (c *Client) ListObjects(ctx context.Context, bucket, prefix string) []minio.ObjectInfo {
	var items []minio.ObjectInfo
	for obj := range c.mc.ListObjects(ctx, bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	}) {
		if obj.Err == nil {
			items = append(items, obj)
		}
	}
	return items
}

// GetObject returns a streaming reader for the named object.
func (c *Client) GetObject(ctx context.Context, bucket, key string) (*minio.Object, error) {
	return c.mc.GetObject(ctx, bucket, key, minio.GetObjectOptions{})
}

// PutObjectStream uploads an object by streaming from reader.
// Pass size = -1 when the content length is unknown; the SDK will use
// multipart upload automatically for large payloads.
func (c *Client) PutObjectStream(ctx context.Context, bucket, key string, reader io.Reader, size int64, contentType string) (minio.UploadInfo, error) {
	return c.mc.PutObject(ctx, bucket, key, reader, size, minio.PutObjectOptions{
		ContentType: contentType,
	})
}

// RemoveObject deletes a single object.
func (c *Client) RemoveObject(ctx context.Context, bucket, key string) error {
	return c.mc.RemoveObject(ctx, bucket, key, minio.RemoveObjectOptions{})
}

// PresignedGetObject returns a presigned URL for downloading an object.
// The URL expires after the given duration (max 7 days for most S3-compatible stores).
func (c *Client) PresignedGetObject(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedGetObject(ctx, bucket, key, expiry, url.Values{})
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// PresignedPutObject returns a presigned URL for uploading an object.
// The URL expires after the given duration (max 7 days for most S3-compatible stores).
func (c *Client) PresignedPutObject(ctx context.Context, bucket, key string, expiry time.Duration) (string, error) {
	u, err := c.mc.PresignedPutObject(ctx, bucket, key, expiry)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

// RemoveObjectsWithPrefix deletes all objects whose key starts with prefix.
func (c *Client) RemoveObjectsWithPrefix(ctx context.Context, bucket, prefix string) error {
	objectsCh := make(chan minio.ObjectInfo)
	go func() {
		defer close(objectsCh)
		for obj := range c.mc.ListObjects(ctx, bucket, minio.ListObjectsOptions{
			Prefix:    prefix,
			Recursive: true,
		}) {
			if obj.Err != nil {
				continue
			}
			objectsCh <- obj
		}
	}()
	for err := range c.mc.RemoveObjects(ctx, bucket, objectsCh, minio.RemoveObjectsOptions{}) {
		if err.Err != nil {
			return err.Err
		}
	}
	return nil
}
