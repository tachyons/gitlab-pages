package disk

import (
	"context"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	contentencoding "gitlab.com/feistel/go-contentencoding/encoding"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

var (
	// Server side content encoding priority.
	supportedEncodings = contentencoding.Preference{
		contentencoding.Brotli:   1.0,
		contentencoding.Gzip:     0.5,
		contentencoding.Identity: 0.1,
	}

	compressedEncodings = map[string]string{
		contentencoding.Brotli: ".br",
		contentencoding.Gzip:   ".gz",
	}
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
func (reader *Reader) detectContentType(ctx context.Context, root vfs.Root, path string) (string, error) {
	contentType := mime.TypeByExtension(filepath.Ext(path))

	if contentType == "" {
		var buf [512]byte

		file, err := root.Open(ctx, path)
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

func (reader *Reader) handleContentEncoding(ctx context.Context, w http.ResponseWriter, r *http.Request, root vfs.Root, fullPath string) string {
	// don't accept range requests for compressed content
	if r.Header.Get("Range") != "" {
		return fullPath
	}

	acceptHeader := r.Header.Get("Accept-Encoding")

	// don't send compressed content if there's no accept-encoding header
	if acceptHeader == "" {
		return fullPath
	}

	results, err := supportedEncodings.Negotiate(acceptHeader, contentencoding.AliasIdentity)
	if err != nil {
		return fullPath
	}

	if len(results) == 0 {
		return fullPath
	}

	for _, encoding := range results {
		if encoding == contentencoding.Identity {
			break
		}

		extension := compressedEncodings[encoding]
		path := fullPath + extension

		// Ensure the file is not a symlink
		if fi, err := root.Lstat(ctx, path); err == nil && fi.Mode().IsRegular() {
			w.Header().Set("Content-Encoding", encoding)

			// http.ServeContent doesn't set Content-Length if Content-Encoding is set
			w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))

			return path
		}
	}

	return fullPath
}
