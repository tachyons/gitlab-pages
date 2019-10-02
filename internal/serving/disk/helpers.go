package disk

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

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
func detectContentType(path string) (string, error) {
	contentType := mime.TypeByExtension(filepath.Ext(path))

	if contentType == "" {
		var buf [512]byte

		file, err := os.Open(path)
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

func handleGZip(w http.ResponseWriter, r *http.Request, fullPath string) string {
	if !acceptsGZip(r) {
		return fullPath
	}

	gzipPath := fullPath + ".gz"

	// Ensure the .gz file is not a symlink
	if fi, err := os.Lstat(gzipPath); err != nil || !fi.Mode().IsRegular() {
		return fullPath
	}

	w.Header().Set("Content-Encoding", "gzip")

	return gzipPath
}
