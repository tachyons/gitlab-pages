package disk

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
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/symlink"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

// Reader is a disk access driver
type Reader struct {
	fileSizeMetric prometheus.Histogram
	vfs            map[string]vfs.VFS
}

func (reader *Reader) tryFile(h serving.Handler) error {
	ctx := h.Request.Context()
	root, fullPath, err := reader.resolvePath(ctx, h.LookupPath, h.SubPath)

	request := h.Request
	host := request.Host
	urlPath := request.URL.Path

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(urlPath) {
			root, fullPath, err = reader.resolvePath(ctx, h.LookupPath, h.SubPath, "index.html")
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
		root, fullPath, err = reader.resolvePath(ctx, h.LookupPath, strings.TrimSuffix(h.SubPath, "/")+".html")
	}

	if err != nil {
		return err
	}

	return reader.serveFile(ctx, h.Writer, h.Request, root, fullPath, h.LookupPath.HasAccessControl)
}

func (reader *Reader) tryNotFound(h serving.Handler) error {
	ctx := h.Request.Context()
	root, page404, err := reader.resolvePath(ctx, h.LookupPath, "404.html")
	if err != nil {
		return err
	}

	err = reader.serveCustomFile(ctx, h.Writer, h.Request, http.StatusNotFound, root, page404)
	if err != nil {
		return err
	}
	return nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (reader *Reader) resolvePath(ctx context.Context, lookupPath *serving.LookupPath, subPath ...string) (vfs.Root, string, error) {
	vfs := reader.vfs[lookupPath.VFS]
	if vfs == nil {
		return nil, "", fmt.Errorf("unknown VFS: %q", lookupPath.VFS)
	}

	root, err := vfs.Root(ctx, lookupPath.Path)
	if err != nil {
		return nil, "", err
	}

	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := strings.Join(subPath, "/")
	fullPath, err := symlink.EvalSymlinks(ctx, root, testPath)

	if err != nil {
		if endsWithoutHTMLExtension(testPath) {
			return nil, "", &locationFileNoExtensionError{
				FullPath: fullPath,
			}
		}

		return nil, "", err
	}

	fi, err := root.Lstat(ctx, fullPath)
	if err != nil {
		return nil, "", err
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return nil, "", &locationDirectoryError{
			FullPath:     fullPath,
			RelativePath: testPath,
		}
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return nil, "", fmt.Errorf("%s: is not a regular file", fullPath)
	}

	return root, fullPath, nil
}

func (reader *Reader) serveFile(ctx context.Context, w http.ResponseWriter, r *http.Request, root vfs.Root, origPath string, accessControl bool) error {
	fullPath := reader.handleGZip(ctx, w, r, root, origPath)

	file, err := root.Open(ctx, fullPath)
	if err != nil {
		return err
	}

	defer file.Close()

	fi, err := root.Lstat(ctx, fullPath)
	if err != nil {
		return err
	}

	if !accessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	contentType, err := reader.detectContentType(ctx, root, origPath)
	if err != nil {
		return err
	}

	reader.fileSizeMetric.Observe(float64(fi.Size()))

	w.Header().Set("Content-Type", contentType)

	// TODO: Support it here
	if rs, ok := file.(io.ReadSeeker); ok {
		http.ServeContent(w, r, origPath, fi.ModTime(), rs)
	} else {
		// Support ReadSeeker if available
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
		io.Copy(w, file)
	}

	return nil
}

func (reader *Reader) serveCustomFile(ctx context.Context, w http.ResponseWriter, r *http.Request, code int, root vfs.Root, origPath string) error {
	fullPath := reader.handleGZip(ctx, w, r, root, origPath)

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
