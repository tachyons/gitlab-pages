package disk

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"gocloud.dev/blob"
	_ "gocloud.dev/blob/fileblob" // Enable local file backend
	"gocloud.dev/gcerrors"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// Reader is a disk access driver
type Reader struct {
	fileSizeMetric prometheus.Histogram
}

type responder struct {
	ctx                       context.Context
	bucket                    *blob.Bucket
	extraDiskPermissionChecks bool
	reader                    *Reader
}

func (reader *Reader) newResponder(h serving.Handler) (*responder, error) {
	resp := &responder{
		ctx:                       h.Request.Context(),
		extraDiskPermissionChecks: true,
		reader:                    reader,
	}
	var err error
	resp.bucket, err = blob.OpenBucket(resp.ctx, "file://.")
	if err != nil {
		return nil, fmt.Errorf("OpenBucket: %v", err)
	}

	go func() {
		<-resp.ctx.Done()
		resp.bucket.Close()
	}()

	return resp, nil
}

func (reader *Reader) tryFile(h serving.Handler) error {
	resp, err := reader.newResponder(h)
	if err != nil {
		return err
	}

	fullPath, err := resp.resolvePath(h.LookupPath.Path, h.SubPath)

	request := h.Request
	host := request.Host
	urlPath := request.URL.Path

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(urlPath) {
			fullPath, err = resp.resolvePath(h.LookupPath.Path, h.SubPath, "index.html")
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
		fullPath, err = resp.resolvePath(h.LookupPath.Path, strings.TrimSuffix(h.SubPath, "/")+".html")
	}

	if err != nil {
		return err
	}

	return resp.serveFile(h.Writer, h.Request, fullPath, h.LookupPath.HasAccessControl)
}

func (reader *Reader) tryNotFound(h serving.Handler) error {
	resp, err := reader.newResponder(h)
	if err != nil {
		return err
	}

	page404, err := resp.resolvePath(h.LookupPath.Path, "404.html")
	if err != nil {
		return err
	}

	err = resp.serveCustomFile(h.Writer, h.Request, http.StatusNotFound, page404)
	if err != nil {
		return err
	}
	return nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (resp *responder) resolvePath(publicPath string, subPath ...string) (string, error) {
	// Ensure that publicPath always ends with "/"
	publicPath = strings.TrimSuffix(publicPath, "/") + "/"

	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := publicPath + strings.Join(subPath, "/")
	fullPath, err := resp.evalSymlink(testPath)
	if err != nil {
		return "", err
	}

	// The requested path resolved to somewhere outside of the public/ directory
	if !strings.HasPrefix(fullPath, publicPath) && fullPath != filepath.Clean(publicPath) {
		return "", fmt.Errorf("%q should be in %q", fullPath, publicPath)
	}

	_, attrsErr := resp.bucket.Attributes(resp.ctx, fullPath)
	if gcerrors.Code(attrsErr) == gcerrors.NotFound && resp.isDir(fullPath) {
		return "", &locationDirectoryError{
			FullPath:     fullPath,
			RelativePath: strings.TrimPrefix(fullPath, publicPath),
		}
	}

	if resp.extraDiskPermissionChecks {
		fi, err := os.Lstat(fullPath)
		// The file exists, but is not a supported type to serve. Perhaps a block
		// special device or something else that may be a security risk.
		if err == nil && !fi.Mode().IsRegular() {
			return "", fmt.Errorf("%s: is not a regular file", fullPath)
		}
	}

	return fullPath, nil
}

func (resp *responder) evalSymlink(testPath string) (string, error) {
	if !resp.extraDiskPermissionChecks {
		return filepath.Clean(testPath), nil
	}

	// This may be a legacy upload where do we want to respect symlinks, but
	// do not want to follow them outside the deployment.
	fullPath, err := filepath.EvalSymlinks(testPath)

	if err != nil && endsWithoutHTMLExtension(testPath) {
		err = &locationFileNoExtensionError{
			FullPath: fullPath,
		}
	}

	return fullPath, err
}

func (resp *responder) isDir(key string) bool {
	iter := resp.bucket.List(&blob.ListOptions{
		Prefix:    key,
		Delimiter: "/",
	})
	obj, err := iter.Next(resp.ctx)
	if err != nil {
		return false
	}

	return obj.Key == key+"/" && obj.IsDir
}

func (resp *responder) serveFile(w http.ResponseWriter, r *http.Request, origPath string, accessControl bool) error {
	fullPath, attrs, err := resp.handleGZip(w, r, origPath)
	if err != nil {
		return err
	}

	if !accessControl {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	contentType, err := resp.detectContentType(origPath)
	if err != nil {
		return err
	}

	resp.reader.fileSizeMetric.Observe(float64(attrs.Size))

	brs := &blobReadSeeker{
		bucket: resp.bucket,
		ctx:    resp.ctx,
		key:    fullPath,
		size:   attrs.Size,
	}
	defer brs.Close()

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, "", attrs.ModTime, brs)

	return nil
}

func (resp *responder) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, origPath string) error {
	fullPath, attrs, err := resp.handleGZip(w, r, origPath)
	if err != nil {
		return err
	}

	contentType, err := resp.detectContentType(origPath)
	if err != nil {
		return err
	}

	resp.reader.fileSizeMetric.Observe(float64(attrs.Size))

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(attrs.Size, 10))
	w.WriteHeader(code)

	if r.Method != "HEAD" {
		blobReader, err := resp.bucket.NewReader(resp.ctx, fullPath, nil)
		if err != nil {
			return err
		}
		defer blobReader.Close()

		_, err = io.CopyN(w, blobReader, attrs.Size)
		return err
	}

	return nil
}

type blobReadSeeker struct {
	r      *blob.Reader
	pos    int64
	bucket *blob.Bucket
	ctx    context.Context
	key    string
	size   int64
}

func (brs *blobReadSeeker) Close() error {
	if brs.r != nil {
		r := brs.r
		brs.r = nil
		return r.Close()
	}

	return nil
}

func (brs *blobReadSeeker) Read(p []byte) (int, error) {
	if brs.r == nil {
		r, err := brs.bucket.NewRangeReader(brs.ctx, brs.key, brs.pos, -1, nil)
		if err != nil {
			return 0, err
		}

		brs.r = r
	}

	return brs.r.Read(p)
}

func (brs *blobReadSeeker) Seek(offset int64, whence int) (int64, error) {
	if brs.r != nil {
		if err := brs.r.Close(); err != nil {
			return 0, err
		}
		brs.r = nil
	}

	switch whence {
	case io.SeekStart:
		brs.pos = offset
	case io.SeekCurrent:
		brs.pos += offset
	case io.SeekEnd:
		brs.pos = brs.size - offset
	}
	if brs.pos < 0 {
		return 0, errors.New("blobReadSeeker: negative seek")
	}

	return brs.pos, nil
}
