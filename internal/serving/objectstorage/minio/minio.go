package minio

import (
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v6"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/objectstorage"
)

// Minio ..
type Minio struct {
	bucket string
	client *minio.Client
}
type object struct {
	object     *minio.Object
	objectInfo *minio.ObjectInfo
}

// New ..
func New(endpoint, bucket, accessKeyID, secretAccessKey string, useSSL bool) (*Minio, error) {
	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}
	return &Minio{
		bucket: bucket,
		client: minioClient,
	}, nil
}

// GetObject ..
func (m *Minio) GetObject(path string) (objectstorage.Object, error) {
	// TODO c.minio.GetObjectWithContext fails locally for some reason
	obj, err := m.client.GetObject(m.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get object: %w", err)
	}
	stat, err := obj.Stat()
	if err != nil {
		if e, ok := err.(minio.ErrorResponse); ok && e.Code == "NoSuchKey" {
			return nil, objectstorage.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to stat object: %w", err)
	}
	return &object{
		object:     obj,
		objectInfo: &stat,
	}, nil
}

// ReaderAt ..
func (o *object) ReaderAt() (io.ReaderAt, error) {
	return o.object, nil
}

// Reader ..
func (o *object) Reader() io.Reader {
	return o.object
}

// Size ..
func (o *object) Size() int64 {
	return o.objectInfo.Size
}

// Name ..
func (o *object) Name() string {
	return o.objectInfo.Key
}

// ModTime ..
func (o *object) ModTime() time.Time {
	return o.objectInfo.LastModified
}

// ContentType ..
func (o *object) ContentType() string {
	contentType := o.objectInfo.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(o.Name()))
	}
	return contentType
}

// Close ..
func (o *object) Close() error {
	return o.object.Close()
}
