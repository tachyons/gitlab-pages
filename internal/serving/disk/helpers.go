package disk

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"gocloud.dev/blob"
	"golang.org/x/sys/unix"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httputil"
)

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

func endsWithoutHTMLExtension(path string) bool {
	return !strings.HasSuffix(path, ".html")
}

func openNoFollow(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW, 0)
}

// Detect file's content-type either by extension or mime-sniffing.
// Implementation is adapted from Golang's `http.serveContent()`
// See https://github.com/golang/go/blob/902fc114272978a40d2e65c2510a18e870077559/src/net/http/fs.go#L194
func (resp *responder) detectContentType(path string) (string, error) {
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType != "" {
		return contentType, nil
	}

	var buf [512]byte
	r, err := resp.bucket.NewRangeReader(resp.ctx, path, 0, int64(len(buf)), nil)
	if err != nil {
		return "", err
	}
	defer r.Close()

	// Using `io.ReadFull()` because `file.Read()` may be chunked.
	// Ignoring errors because we don't care if the 512 bytes cannot be read.
	n, _ := io.ReadFull(r, buf[:])
	contentType = http.DetectContentType(buf[:n])

	return contentType, nil
}

func acceptsGZip(r *http.Request) bool {
	if r.Header.Get("Range") != "" {
		return false
	}

	offers := []string{"gzip", "identity"}
	acceptedEncoding := httputil.NegotiateContentEncoding(r, offers)
	return acceptedEncoding == "gzip"
}

func (resp *responder) handleGZip(w http.ResponseWriter, r *http.Request, key string) (string, *blob.Attributes, error) {
	attrs, err := resp.bucket.Attributes(resp.ctx, key)
	if err != nil {
		return "", nil, err
	}

	if !acceptsGZip(r) {
		return key, attrs, nil
	}

	gzipPath := key + ".gz"
	gzipAttrs, err := resp.bucket.Attributes(resp.ctx, gzipPath)
	if err != nil {
		return key, attrs, nil
	}

	if resp.disk {
		// Ensure the .gz file is not a symlink
		if fi, err := os.Lstat(gzipPath); err != nil || !fi.Mode().IsRegular() {
			return key, attrs, nil
		}
	}

	w.Header().Set("Content-Encoding", "gzip")

	return gzipPath, gzipAttrs, nil
}
