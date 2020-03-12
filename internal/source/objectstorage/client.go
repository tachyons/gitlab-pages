package objectstorage

import (
	"errors"
	"fmt"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v6"
	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// ErrKeyNotFound TODO update doc
var ErrKeyNotFound = errors.New("key not found")

type Client struct {
	bucket string
	minio  *minio.Client
}

func New(endpoint, bucket, accessKeyID, secretAccessKey string, useSSL bool) (*Client, error) {
	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}
	return &Client{
		bucket: bucket,
		minio:  minioClient,
	}, nil
}

// func (c *Client) ListBucketsAndObjects() {
// 	buckets, err := c.minio.ListBuckets()
// 	if err != nil {
// 		fmt.Println(err)
// 		return
// 	}
// 	doneCh := make(chan struct{})
// 	defer close(doneCh)
//
// 	for k, bucket := range buckets {
// 		fmt.Printf("%d - %+v\n\n", k+1, bucket)
// 		if bucket.Name == "pages" {
// 			for obj := range c.minio.ListObjectsV2WithMetadata(bucket.Name, "root/blog/public/", true, doneCh) {
// 				fmt.Printf("found a keeeeey: %q", obj.Key)
// 			}
// 		}
// 	}
// }
func (c *Client) GetObject(path string) (*minio.Object, *minio.ObjectInfo, error) {
	// TODO c.minio.GetObjectWithContext fails locally for some reason
	obj, err := c.minio.GetObject(c.bucket, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get object: %w", err)
	}
	stat, err := obj.Stat()
	if err != nil {
		if e, ok := err.(minio.ErrorResponse); ok && e.Code == "NoSuchKey" {
			return nil, nil, ErrKeyNotFound
		}
		return nil, nil, fmt.Errorf("failed to stat object: %w", err)
	}
	return obj, &stat, nil
}

func (c *Client) ServeFileHTTP(handler serving.Handler) bool {
	// TODO validate different paths like disk/Reader.tryFile
	fullPath := strings.TrimSuffix(handler.LookupPath.Path, "/") + "/"
	pathWithoutExt := fullPath

	if handler.SubPath != "" {
		fullPath += handler.SubPath
	}

	if endsWithSlash(fullPath) {
		fullPath += "index.html"
	}

	w := handler.Writer
	content, stat, err := c.tryZipFileFirst(pathWithoutExt + "doesnotexist.zip")
	if err != nil {
		log.WithError(err).Error("can't get zip file?")
		return false
	}

	fmt.Printf("we are trying to serve: %q\n", fullPath)
	if content == nil {
		// didn't find zip, try full path
		content, stat, err = c.GetObject(fullPath)
		if err != nil {
			log.WithError(err).Error("failed to get object")
			return false
		}
	}
	defer content.Close()

	if !handler.LookupPath.HasAccessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}
	contentType := mime.TypeByExtension(filepath.Ext(stat.Key))
	fmt.Printf("from object storage: filext  %q vs object stat: %q\n", contentType, stat.ContentType)
	w.Header().Set("Content-Type", contentType)

	fmt.Printf("were about to serve: %q\n%q\n", stat.Key, stat.ContentType)
	// TODO get time object was modified?
	http.ServeContent(w, handler.Request, fullPath, stat.LastModified, content)

	return true
}

func (c *Client) ServeNotFoundHTTP(handler serving.Handler) {
	httperrors.Serve404(handler.Writer)
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

func (c *Client) tryZipFileFirst(path string) (*minio.Object, *minio.ObjectInfo, error) {
	obj, stat, err := c.GetObject(path)
	if err != nil && err == ErrKeyNotFound {
		// could not find zip file
		return nil, nil, nil
	}
	return obj, stat, err
}
