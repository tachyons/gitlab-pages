package disk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gitlab.com/gitlab-org/labkit/errortracking"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/redirects"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/symlink"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	vfsServing "gitlab.com/gitlab-org/gitlab-pages/internal/vfs/serving"
)

// Reader is a disk access driver
type Reader struct {
	fileSizeMetric *prometheus.HistogramVec
	vfs            vfs.VFS
}

// Show the user some validation messages for their _redirects file
func (reader *Reader) serveRedirectsStatus(h serving.Handler, redirects *redirects.Redirects) {
	h.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	h.Writer.Header().Set("X-Content-Type-Options", "nosniff")
	h.Writer.WriteHeader(http.StatusOK)
	fmt.Fprintln(h.Writer, redirects.Status())
}

// tryRedirects returns true if it successfully handled request
func (reader *Reader) tryRedirects(h serving.Handler) bool {
	ctx := h.Request.Context()
	root, err := reader.vfs.Root(ctx, h.LookupPath.Path, h.LookupPath.SHA256)
	if err != nil {
		return handleRootError(err, h)
	}

	r := redirects.ParseRedirects(ctx, root)

	rewrittenURL, status, err := r.Rewrite(h.Request.URL)
	if err != nil {
		if err != redirects.ErrNoRedirect {
			// We assume that rewrite failure is not fatal
			// and we only capture the error
			errortracking.Capture(err, errortracking.WithRequest(h.Request), errortracking.WithStackTrace())
		}
		return false
	}

	if status == http.StatusOK {
		h.SubPath = strings.TrimPrefix(rewrittenURL.Path, h.LookupPath.Prefix)
		return reader.tryFile(h)
	}

	http.Redirect(h.Writer, h.Request, rewrittenURL.Path, status)
	return true
}

// tryFile returns true if it successfully handled request
func (reader *Reader) tryFile(h serving.Handler) bool {
	ctx := h.Request.Context()

	root, err := reader.vfs.Root(ctx, h.LookupPath.Path, h.LookupPath.SHA256)
	if err != nil {
		return handleRootError(err, h)
	}

	fullPath, err := reader.resolvePath(ctx, root, h.SubPath)

	request := h.Request
	urlPath := request.URL.Path

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(urlPath) {
			fullPath, err = reader.resolvePath(ctx, root, h.SubPath, "index.html")
		} else {
			http.Redirect(h.Writer, h.Request, redirectPath(h.Request), http.StatusFound)
			return true
		}
	}

	if locationError, _ := err.(*locationFileNoExtensionError); locationError != nil {
		fullPath, err = reader.resolvePath(ctx, root, strings.TrimSuffix(h.SubPath, "/")+".html")
	}

	if err != nil {
		// We assume that this is mostly missing file type of the error
		// and additional handlers should try to process the request
		return false
	}

	// Serve status of `_redirects` under `_redirects`
	// We check if the final resolved path is `_redirects` after symlink traversal
	if fullPath == redirects.ConfigFile {
		if os.Getenv("FF_ENABLE_REDIRECTS") != "false" {
			r := redirects.ParseRedirects(ctx, root)
			reader.serveRedirectsStatus(h, r)
			return true
		}

		h.Writer.WriteHeader(http.StatusForbidden)
		return true
	}

	return reader.serveFile(ctx, h.Writer, h.Request, root, fullPath, h.LookupPath.HasAccessControl)
}

func redirectPath(request *http.Request) string {
	url := *request.URL

	// This ensures that path starts with `//<host>/`
	url.Scheme = ""
	url.Host = request.Host
	url.Path = strings.TrimPrefix(url.Path, "/") + "/"

	return strings.TrimSuffix(url.String(), "?")
}

func (reader *Reader) tryNotFound(h serving.Handler) bool {
	ctx := h.Request.Context()

	root, err := reader.vfs.Root(ctx, h.LookupPath.Path, h.LookupPath.SHA256)
	if err != nil {
		return handleRootError(err, h)
	}

	page404, err := reader.resolvePath(ctx, root, "404.html")
	if err != nil {
		// We assume that this is mostly missing file type of the error
		// and additional handlers should try to process the request
		return false
	}

	err = reader.serveCustomFile(ctx, h.Writer, h.Request, http.StatusNotFound, root, page404)
	if err != nil {
		// Handle context.Canceled error as not exist https://gitlab.com/gitlab-org/gitlab-pages/-/issues/669
		if errors.Is(err, context.Canceled) {
			logging.LogRequest(h.Request).WithError(err).Warn("user cancelled request")
			return false
		}

		httperrors.Serve500WithRequest(h.Writer, h.Request, "serveCustomFile", err)
		return true
	}

	return true
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (reader *Reader) resolvePath(ctx context.Context, root vfs.Root, subPath ...string) (string, error) {
	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := strings.Join(subPath, "/")
	fullPath, err := symlink.EvalSymlinks(ctx, root, testPath)

	if err != nil {
		if endsWithoutHTMLExtension(testPath) {
			return "", &locationFileNoExtensionError{
				FullPath: fullPath,
			}
		}

		return "", err
	}

	fi, err := root.Lstat(ctx, fullPath)
	if err != nil {
		return "", err
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return "", &locationDirectoryError{
			FullPath:     fullPath,
			RelativePath: testPath,
		}
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s: is not a regular file", fullPath)
	}

	return fullPath, nil
}

func (reader *Reader) serveFile(ctx context.Context, w http.ResponseWriter, r *http.Request, root vfs.Root, origPath string, accessControl bool) bool {
	fullPath := reader.handleContentEncoding(ctx, w, r, root, origPath)

	file, err := root.Open(ctx, fullPath)
	if err != nil {
		httperrors.Serve500WithRequest(w, r, "root.Open", err)
		return true
	}

	defer file.Close()

	fi, err := root.Lstat(ctx, fullPath)
	if err != nil {
		httperrors.Serve500WithRequest(w, r, "root.Lstat", err)
		return true
	}

	if !accessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	contentType, err := reader.detectContentType(ctx, root, origPath)
	if err != nil {
		httperrors.Serve500WithRequest(w, r, "detectContentType", err)
		return true
	}

	w.Header().Set("Content-Type", contentType)

	reader.fileSizeMetric.WithLabelValues(reader.vfs.Name()).Observe(float64(fi.Size()))

	// Support vfs.SeekableFile if available (uncompressed files)
	if rs, ok := file.(vfs.SeekableFile); ok {
		http.ServeContent(w, r, origPath, fi.ModTime(), rs)
	} else {
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
		vfsServing.ServeCompressedFile(w, r, fi.ModTime(), file)
	}

	return true
}

func (reader *Reader) serveCustomFile(ctx context.Context, w http.ResponseWriter, r *http.Request, code int, root vfs.Root, origPath string) error {
	fullPath := reader.handleContentEncoding(ctx, w, r, root, origPath)

	// Open and serve content of file
	file, err := root.Open(ctx, fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := root.Lstat(ctx, fullPath)
	if err != nil {
		return err
	}

	contentType, err := reader.detectContentType(ctx, root, origPath)
	if err != nil {
		return err
	}

	reader.fileSizeMetric.WithLabelValues(reader.vfs.Name()).Observe(float64(fi.Size()))

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(code)

	if r.Method != "HEAD" {
		_, err := io.CopyN(w, file, fi.Size())
		return err
	}

	return nil
}

func handleRootError(err error, h serving.Handler) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, fs.ErrNotExist) {
		return false
	}

	if errors.Is(err, context.Canceled) {
		// Handle context.Canceled error as not found exist https://gitlab.com/gitlab-org/gitlab-pages/-/issues/669
		httperrors.Serve404(h.Writer)
		return true
	}

	httperrors.Serve500WithRequest(h.Writer, h.Request, "vfs.Root", err)
	return true

}
