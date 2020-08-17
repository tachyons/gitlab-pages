package file

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/file/symlink"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

// Reader is a disk access driver
type Reader struct {
	fileSizeMetric prometheus.Histogram
	vfs            vfs.VFS
}

func (reader *Reader) tryFile(h serving.Handler) error {
	ctx := h.Request.Context()
	dir, fullPath, err := reader.resolvePath(ctx, h.LookupPath.Path, h.SubPath)

	request := h.Request
	host := request.Host
	urlPath := request.URL.Path

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(urlPath) {
			dir, fullPath, err = reader.resolvePath(ctx, h.LookupPath.Path, h.SubPath, "index.html")
		} else {
			// TODO why are we doing that? In tests it redirects to HTTPS. This seems wrong,
			// issue about this: https://gitlab.com/gitlab-org/gitlab-pages/issues/273

			// Concat Host with URL.Path
			redirectPath := "//" + host + "/"
			redirectPath += strings.TrimPrefix(urlPath, "/")

			// Ensure that there's always "/" at end
			redirectPath = strings.TrimSuffix(redirectPath, "/") + "/"
			http.Redirect(h.Writer, h.Request, redirectPath, 302)
			return nil
		}
	}

	if locationError, _ := err.(*locationFileNoExtensionError); locationError != nil {
		dir, fullPath, err = reader.resolvePath(ctx, h.LookupPath.Path, strings.TrimSuffix(h.SubPath, "/")+".html")
	}

	if err != nil {
		return err
	}

	return reader.serveFile(ctx, h.Writer, h.Request, dir, fullPath, h.LookupPath.HasAccessControl)
}

func (reader *Reader) tryNotFound(h serving.Handler) error {
	ctx := h.Request.Context()
	dir, page404, err := reader.resolvePath(ctx, h.LookupPath.Path, "404.html")
	if err != nil {
		return err
	}

	err = reader.serveCustomFile(ctx, h.Writer, h.Request, http.StatusNotFound, dir, page404)
	if err != nil {
		return err
	}
	return nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (reader *Reader) resolvePath(ctx context.Context, publicPath string, subPath ...string) (vfs.Dir, string, error) {
	// Ensure that publicPath always ends with "/"
	publicPath = strings.TrimSuffix(publicPath, "/") + "/"

	dir, err := reader.vfs.Dir(ctx, publicPath)
	if err != nil {
		return nil, "", err
	}

	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := strings.Join(subPath, "/")
	fullPath, err := symlink.EvalSymlinks(ctx, dir, testPath)

	if err != nil {
		if endsWithoutHTMLExtension(testPath) {
			return nil, "", &locationFileNoExtensionError{
				FullPath: fullPath,
			}
		}

		return nil, "", err
	}

	fi, err := dir.Lstat(ctx, fullPath)
	if err != nil {
		return nil, "", err
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return nil, "", &locationDirectoryError{
			FullPath:     fullPath,
			RelativePath: strings.TrimPrefix(fullPath, publicPath),
		}
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%s: is not a regular file", fullPath)
	}

	return dir, fullPath, nil
}

func (reader *Reader) serveFile(ctx context.Context, w http.ResponseWriter, r *http.Request, dir vfs.Dir, origPath string, accessControl bool) error {
	fullPath := reader.handleGZip(ctx, w, r, dir, origPath)

	file, err := dir.Open(ctx, fullPath)
	if err != nil {
		return err
	}

	defer file.Close()

	fi, err := dir.Lstat(ctx, fullPath)
	if err != nil {
		return err
	}

	if !accessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	contentType, err := reader.detectContentType(ctx, dir, origPath)
	if err != nil {
		return err
	}

	reader.fileSizeMetric.Observe(float64(fi.Size()))

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, origPath, fi.ModTime(), file)

	return nil
}

func (reader *Reader) serveCustomFile(ctx context.Context, w http.ResponseWriter, r *http.Request, code int, dir vfs.Dir, origPath string) error {
	fullPath := reader.handleGZip(ctx, w, r, dir, origPath)

	// Open and serve content of file
	file, err := dir.Open(ctx, fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := dir.Lstat(ctx, fullPath)
	if err != nil {
		return err
	}

	contentType, err := reader.detectContentType(ctx, dir, origPath)
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
