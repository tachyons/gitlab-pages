package objectstorage

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/zipartifacts"
	"gitlab.com/gitlab-org/gitlab-pages/internal/zipartifacts/reader"
)

type cache interface {
	Set(ctx context.Context, cancel func(), reader *reader.Reader)
	Reader() (*reader.Reader, error)
}

// ObjecStorage implements the serving.Serving interface.
// Contains a cache to store zip.Reader
type ObjectStorage struct {
	cache cache
}

// New gets called 6+ times per request so need to just init once.
// Think about reworking during https://gitlab.com/gitlab-org/gitlab-pages/-/issues/371
var oss = &ObjectStorage{}

func New() *ObjectStorage {
	if oss.cache == nil {
		oss.cache = newInMemoryCache()
	}

	return oss
}

func (os *ObjectStorage) ServeFileHTTP(handler serving.Handler) bool {
	zipReader, err := os.getOrSetReader(handler)
	if err != nil {
		logrus.WithError(err).Error("failed so serve from zip file")
		os.ServeNotFoundHTTP(handler)
		return true
	}

	// TODO implement the logic from the disk reader
	filename := handler.SubPath
	if filename == "" || filename == "/" {
		filename = "index.html"
	}

	err = os.handleZipFile(zipReader, filename, handler, http.StatusOK)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			os.ServeNotFoundHTTP(handler)
			return true
		}
		// TODO add metrics
		logrus.WithError(err).Error("failed to serve from zip file")
		return false
	}

	return true
}

func (os *ObjectStorage) ServeNotFoundHTTP(handler serving.Handler) {
	zipReader, err := os.getOrSetReader(handler)
	if err != nil {
		logrus.WithError(err).Error("failed so serve from zip file")
		httperrors.Serve404(handler.Writer)
		return
	}

	// try to serve custom 404
	err = os.handleZipFile(zipReader, "404.html", handler, http.StatusNotFound)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			logrus.WithError(err).Error("failed to read zip file")
		}

		httperrors.Serve404(handler.Writer)
		return
	}
}

func (os *ObjectStorage) getOrSetReader(handler serving.Handler) (*reader.Reader, error) {
	zipReader, err := os.cache.Reader()
	if err != nil {
		// let the context be canceled on the timeout so that the zipReader stays open for a while
		// this context is used by the os.cache and zipartifacts.OpenArchive
		// TODO configure this timeout
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)

		zipReader, err = zipartifacts.OpenArchive(ctx, handler.LookupPath.Path)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to open zip archive: %v", err)
		}
		go os.cache.Set(ctx, cancel, zipReader)
	}

	return zipReader, nil
}

func (os *ObjectStorage) handleZipFile(reader *reader.Reader, filename string, handler serving.Handler, status int) error {
	file, stat, err := reader.Open(filename)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	contentType := mime.TypeByExtension(filepath.Ext(stat.Name()))
	return writeContent(handler, file, stat.ModTime(), contentType, status)
}

func writeContent(handler serving.Handler, content io.ReadCloser, modTime time.Time, contentType string, status int) error {
	if content == nil {
		return fmt.Errorf("content is nil")
	}

	w := handler.Writer
	w.WriteHeader(status)
	if !handler.LookupPath.HasAccessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Last-Modified", modTime.UTC().Format(http.TimeFormat))

	var err error
	_, err = io.Copy(w, content)
	if err != nil {
		return fmt.Errorf("failed to write response: %v", err)
	}

	return nil
}
