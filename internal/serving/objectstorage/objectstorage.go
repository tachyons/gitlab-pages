package objectstorage

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v6"
	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/zip"
)

// ErrKeyNotFound TODO update doc
var ErrKeyNotFound = errors.New("key not found")

type Client struct {
	bucket string
	minio  *minio.Client

	// TODO: cache zip files by projectID for now, will need to expire/cleanup
	cacheMux      sync.Mutex
	cachedReaders map[uint64]*zip.Reader
}

func New(endpoint, bucket, accessKeyID, secretAccessKey string, useSSL bool) (*Client, error) {
	minioClient, err := minio.New(endpoint, accessKeyID, secretAccessKey, useSSL)
	if err != nil {
		return nil, fmt.Errorf("failed to create minio client: %w", err)
	}
	return &Client{
		bucket:        bucket,
		minio:         minioClient,
		cacheMux:      sync.Mutex{},
		cachedReaders: map[uint64]*zip.Reader{},
	}, nil
}

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
	served, err := c.tryZipFile(handler)
	if err != nil {
		log.WithError(err).Error("file not found in archive")
		return false
	}

	if !served {
		if err := c.serveFile(handler); err != nil {
			log.WithError(err).Error("can't serve file")
			return false
		}
	}

	return true
}

func (c *Client) ServeNotFoundHTTP(handler serving.Handler) {
	httperrors.Serve404(handler.Writer)
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

func (c *Client) serveFile(handler serving.Handler) error {
	// TODO validate different paths like disk/Reader.tryFile
	fileName := strings.TrimSuffix(handler.LookupPath.Path, "/") + "/"
	if handler.SubPath != "" {
		fileName += handler.SubPath
	}

	if endsWithSlash(fileName) {
		fileName += "index.html"
	}
	content, stat, err := c.GetObject(fileName)
	if err != nil {
		return err
	}
	return writeContent(handler, content, stat.Key, stat.LastModified, false)
}

func (c *Client) tryZipFile(handler serving.Handler) (bool, error) {
	projectID := handler.LookupPath.ProjectID
	c.cacheMux.Lock()
	reader, ok := c.cachedReaders[projectID]
	c.cacheMux.Unlock()
	if !ok {
		// TODO assume the API returns the base path of the project and we'll look for artifact.zip
		obj, objStat, err := c.GetObject(handler.LookupPath.Path + "artifact.zip")
		if err != nil {
			if err == ErrKeyNotFound {
				// could not find zip file
				return false, nil
			}
			return false, fmt.Errorf("failed to get object: %w", err)
		}
		// override reader
		reader, err = zip.New(obj, objStat.Size)
		if err != nil {
			return false, fmt.Errorf("failed create zip.Reader: %w", err)
		}
		c.cacheMux.Lock()
		c.cachedReaders[projectID] = reader
		c.cacheMux.Unlock()
	}

	filename := "index.html"
	if handler.SubPath != "" {
		filename = handler.SubPath
	}

	file, stat, err := reader.Open(filename)
	if err != nil {
		return false, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	err = writeContent(handler, file, stat.Name(), stat.ModTime(), true)
	return err == nil, err
}

func writeContent(handler serving.Handler, content interface{}, fileName string, modTime time.Time, fromZip bool) error {
	w := handler.Writer
	if !handler.LookupPath.HasAccessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}
	contentType := mime.TypeByExtension(filepath.Ext(fileName))
	w.Header().Set("Content-Type", contentType)

	// TODO implement Seek(offset int64, whence int) (int64, error) so that we can use http.ServeContent?
	var err error
	if fromZip {
		_, err = io.Copy(w, content.(io.Reader))
		if err != nil {
			return fmt.Errorf("failed to write response: %w", err)
		}
	} else {
		http.ServeContent(w, handler.Request, fileName, modTime, content.(io.ReadSeeker))
	}
	return nil
}
