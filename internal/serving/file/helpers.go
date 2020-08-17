package file

import (
	"context"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httputil"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

func endsWithoutHTMLExtension(path string) bool {
	return !strings.HasSuffix(path, ".html")
}

// Detect file's content-type either by extension or mime-sniffing.
// Implementation is adapted from Golang's `http.serveContent()`
// See https://github.com/golang/go/blob/902fc114272978a40d2e65c2510a18e870077559/src/net/http/fs.go#L194
func (reader *Reader) detectContentType(ctx context.Context, dir vfs.Dir, path string) (string, error) {
	contentType := mime.TypeByExtension(filepath.Ext(path))

	if contentType == "" {
		var buf [512]byte

		file, err := dir.Open(ctx, path)
		if err != nil {
			return "", err
		}

		defer file.Close()

		// Using `io.ReadFull()` because `file.Read()` may be chunked.
		// Ignoring errors because we don't care if the 512 bytes cannot be read.
		n, _ := io.ReadFull(file, buf[:])
		contentType = http.DetectContentType(buf[:n])
	}

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

func (reader *Reader) handleGZip(ctx context.Context, w http.ResponseWriter, r *http.Request, dir vfs.Dir, fullPath string) string {
	if !acceptsGZip(r) {
		return fullPath
	}

	gzipPath := fullPath + ".gz"

	// Ensure the .gz file is not a symlink
	fi, err := dir.Lstat(ctx, gzipPath)
	if err != nil || !fi.Mode().IsRegular() {
		return fullPath
	}

	w.Header().Set("Content-Encoding", "gzip")

	// http.ServeContent doesn't set Content-Length if Content-Encoding is set
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))

	return gzipPath
}
