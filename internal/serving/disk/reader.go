package disk

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/fileresolver"
)

// Reader is a disk access driver
type Reader struct {
	fileSizeMetric prometheus.Histogram
}

func (reader *Reader) tryFile(h serving.Handler) error {
	fullPath, err := reader.resolvePath(h.LookupPath.Path, h.SubPath, h.Request.URL.Path)
	if err != nil {
		return err
	}

	return reader.serveFile(h.Writer, h.Request, fullPath, h.LookupPath.HasAccessControl)
}

func (reader *Reader) tryNotFound(h serving.Handler) error {
	page404, err := reader.resolvePath(h.LookupPath.Path, "", "404.html")
	if err != nil {
		return err
	}

	err = reader.serveCustomFile(h.Writer, h.Request, http.StatusNotFound, page404)
	if err != nil {
		return err
	}
	return nil
}

// redirect for css files  root.gitlab.io/mywebsite
// if dir serve from there, it's trick

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (reader *Reader) resolvePath(publicPath, subPath, urlPath string) (string, error) {
	fullPath, err := fileresolver.ResolveFilePath(publicPath, subPath, urlPath, filepath.EvalSymlinks)
	if err != nil {
		return "", err
	}

	fi, err := os.Lstat(fullPath)
	if err != nil {
		return "", err
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return "", &locationDirectoryError{
			FullPath:     fullPath,
			RelativePath: strings.TrimPrefix(fullPath, publicPath),
		}
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s: is not a regular file", fullPath)
	}

	return fullPath, nil
}

func (reader *Reader) serveFile(w http.ResponseWriter, r *http.Request, origPath string, accessControl bool) error {
	fullPath := handleGZip(w, r, origPath)

	file, err := openNoFollow(fullPath)
	if err != nil {
		return err
	}

	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	if !accessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	contentType, err := detectContentType(origPath)
	if err != nil {
		return err
	}

	reader.fileSizeMetric.Observe(float64(fi.Size()))

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, origPath, fi.ModTime(), file)

	return nil
}

func (reader *Reader) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, origPath string) error {
	fullPath := handleGZip(w, r, origPath)

	// Open and serve content of file
	file, err := openNoFollow(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	contentType, err := detectContentType(origPath)
	if err != nil {
		return err
	}

	reader.fileSizeMetric.Observe(float64(fi.Size()))

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(code)

	if r.Method != "HEAD" {
		_, err := io.CopyN(w, file, fi.Size())
		return err
	}

	return nil
}
