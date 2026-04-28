package api

import (
	"fmt"
	"io"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/craftslab/s3c/backend/storage"
	"github.com/gin-gonic/gin"
)

// Handler holds the dependencies for all HTTP handlers.
type Handler struct {
	client *storage.Client
}

// ----- bucket handlers -----

// ListBuckets returns all buckets as JSON.
func (h *Handler) ListBuckets(c *gin.Context) {
	buckets, err := h.client.ListBuckets(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, buckets)
}

// CreateBucket creates a new bucket.
func (h *Handler) CreateBucket(c *gin.Context) {
	var req struct {
		Name   string `json:"name" binding:"required"`
		Region string `json:"region"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Region == "" {
		req.Region = "us-east-1"
	}
	if err := h.client.MakeBucket(c.Request.Context(), req.Name, req.Region); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"message": "bucket created"})
}

// DeleteBucket removes a bucket.
func (h *Handler) DeleteBucket(c *gin.Context) {
	bucket := c.Param("bucket")
	if err := h.client.RemoveBucket(c.Request.Context(), bucket); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "bucket deleted"})
}

// ----- object handlers -----

// ObjectItem is the JSON representation of a single S3 object or prefix.
type ObjectItem struct {
	Key          string    `json:"key"`
	Name         string    `json:"name"`
	Size         int64     `json:"size"`
	LastModified time.Time `json:"lastModified"`
	IsDir        bool      `json:"isDir"`
	ContentType  string    `json:"contentType"`
	ETag         string    `json:"etag"`
}

// ListObjects lists objects in a bucket under an optional prefix.
func (h *Handler) ListObjects(c *gin.Context) {
	bucket := c.Param("bucket")
	prefix := c.Query("prefix")

	raw := h.client.ListObjects(c.Request.Context(), bucket, prefix)
	items := make([]ObjectItem, 0, len(raw))
	for _, obj := range raw {
		isDir := strings.HasSuffix(obj.Key, "/")
		name := path.Base(strings.TrimSuffix(obj.Key, "/"))
		items = append(items, ObjectItem{
			Key:          obj.Key,
			Name:         name,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			IsDir:        isDir,
			ContentType:  obj.ContentType,
			ETag:         obj.ETag,
		})
	}
	c.JSON(http.StatusOK, items)
}

// DownloadObject streams an S3 object directly to the HTTP response.
// The file is never fully buffered in memory; data flows from S3 → client.
func (h *Handler) DownloadObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := strings.TrimPrefix(c.Param("key"), "/")

	obj, err := h.client.GetObject(c.Request.Context(), bucket, key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer obj.Close()

	info, err := obj.Stat()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	contentType := info.ContentType
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	c.DataFromReader(http.StatusOK, info.Size, contentType, obj, map[string]string{
		"Content-Disposition": fmt.Sprintf(`attachment; filename="%s"`, path.Base(key)),
	})
}

// UploadObject handles one or more file uploads, streaming each file part
// directly to S3 without buffering the entire payload to disk.
func (h *Handler) UploadObject(c *gin.Context) {
	bucket := c.Param("bucket")
	prefix := c.Query("prefix")

	mr, err := c.Request.MultipartReader()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid multipart request: " + err.Error()})
		return
	}

	var uploaded []gin.H
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		filename := part.FileName()
		if filename == "" {
			// skip non-file fields
			continue
		}

		key := filename
		if prefix != "" {
			key = strings.TrimSuffix(prefix, "/") + "/" + filename
		}

		contentType := part.Header.Get("Content-Type")
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// size=-1 tells the MinIO SDK to use multipart upload transparently,
		// which is the correct strategy for large files.
		if _, err := h.client.PutObjectStream(c.Request.Context(), bucket, key, part, -1, contentType); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		uploaded = append(uploaded, gin.H{"key": key, "name": filename})
	}

	if len(uploaded) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no files found in request"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"uploaded": uploaded})
}

// DeleteObject removes a single object or, when the key ends with "/",
// recursively removes all objects under that prefix.
func (h *Handler) DeleteObject(c *gin.Context) {
	bucket := c.Param("bucket")
	key := strings.TrimPrefix(c.Param("key"), "/")

	if strings.HasSuffix(key, "/") {
		if err := h.client.RemoveObjectsWithPrefix(c.Request.Context(), bucket, key); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	} else {
		if err := h.client.RemoveObject(c.Request.Context(), bucket, key); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// parseExpiry reads the "expiry" query parameter (in seconds) and returns a
// time.Duration.  When the parameter is absent or invalid the default of 24 h
// is used.  The maximum accepted value is 7 days (604 800 s).
func parseExpiry(c *gin.Context) time.Duration {
	const defaultExpiry = 24 * time.Hour
	const maxExpiry = 7 * 24 * time.Hour

	s := c.Query("expiry")
	if s == "" {
		return defaultExpiry
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil || v <= 0 {
		return defaultExpiry
	}
	d := time.Duration(v) * time.Second
	if d > maxExpiry {
		d = maxExpiry
	}
	return d
}

// GenerateDownloadLink returns a presigned GET URL for a specific object.
// Query param: expiry (seconds, default 86400, max 604800).
func (h *Handler) GenerateDownloadLink(c *gin.Context) {
	bucket := c.Param("bucket")
	key := strings.TrimPrefix(c.Param("key"), "/")

	expiry := parseExpiry(c)
	u, err := h.client.PresignedGetObject(c.Request.Context(), bucket, key, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        u,
		"expires_in": int64(expiry.Seconds()),
	})
}

// GenerateUploadLink returns a presigned PUT URL for uploading to a specific key.
// Query param: expiry (seconds, default 86400, max 604800).
func (h *Handler) GenerateUploadLink(c *gin.Context) {
	bucket := c.Param("bucket")
	key := strings.TrimPrefix(c.Param("key"), "/")

	expiry := parseExpiry(c)
	u, err := h.client.PresignedPutObject(c.Request.Context(), bucket, key, expiry)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"url":        u,
		"key":        key,
		"expires_in": int64(expiry.Seconds()),
	})
}
