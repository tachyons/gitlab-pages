package gcs

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"path/filepath"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/objectstorage"
)

const defaultAppCreds = "/Users/jaime/.gitlab/gitlab-gcp-creds.json"
const bucketName = "jaime-test-bucket"

var (
	defaultTimeout = time.Second * 10
)

// GCS ..
type GCS struct {
	bucket string
	client *storage.Client
}

type object struct {
	reader    *storage.Reader
	objHandle *storage.ObjectHandle
}

// NewGCS ..
func NewGCS(bucket string) (*GCS, error) {
	// Creates the new bucket.
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Creates a client.
	client, err := storage.NewClient(ctx,
		option.WithCredentialsFile(defaultAppCreds),
		option.WithScopes(storage.ScopeReadOnly))
	if err != nil {
		return nil, fmt.Errorf("failed to create gcs client: %v", err)
	}
	return &GCS{
		bucket: bucket,
		client: client,
	}, nil
}

// GetObject ..
func (gcs *GCS) GetObject(path string) (objectstorage.Object, error) {
	// TODO make this context cancellable from the caller
	ctx := context.Background()

	objHandle := gcs.client.Bucket(gcs.bucket).Object(path)
	reader, err := objHandle.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, objectstorage.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get reader... %v", err)
	}

	return &object{
		reader:    reader,
		objHandle: objHandle,
	}, nil
}

// ReaderAt ..
func (o *object) ReaderAt() (io.ReaderAt, error) {
	buff := bytes.NewBuffer([]byte{})
	_, err := io.Copy(buff, o.reader)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buff.Bytes()), nil
}

// Reader ..
func (o *object) Reader() io.Reader {
	return o.reader
}

// Size ..
func (o *object) Size() int64 {
	return o.reader.Attrs.Size
}

// Name ..
func (o *object) Name() string {
	return o.objHandle.ObjectName()
}

// ModTime ..
func (o *object) ModTime() time.Time {
	return o.reader.Attrs.LastModified
}

// ContentType ..
func (o *object) ContentType() string {
	contentType := o.reader.Attrs.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(o.Name()))
	}
	return contentType
}

// Close ..
func (o *object) Close() error {
	return o.reader.Close()
}
