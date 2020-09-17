package disk

import (
	"context"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httputil"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
)

var compressedEncodings = map[string]string{
	"br":   ".br",
	"gzip": ".gz",
}

// Server side content encoding priority.
// Map iteration order is not deterministic in go, so we need this array to specify the priority
// when the client doesn't provide one
var compressedEncodingsPriority = []string{
	"br",
	"gzip",
}

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
	fileExt := filepath.Ext(path)
	contentType := mime.TypeByExtension(fileExt)

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

	// see https://gitlab.com/gitlab-org/gitlab-pages/-/issues/460
	// packages mime and http currently do not know about the avif file format
	if contentType == "application/octet-stream" && fileExt == ".avif" {
		contentType = "image/avif"
	}

	return contentType, nil
}

func (reader *Reader) handleContentEncoding(ctx context.Context, w http.ResponseWriter, r *http.Request, root vfs.Root, fullPath string) string {
	// don't accept range requests for compressed content
	if r.Header.Get("Range") != "" {
		return fullPath
	}

	files := map[string]os.FileInfo{}

	// finding compressed files
	for encoding, extension := range compressedEncodings {
		path := fullPath + extension

		// Ensure the file is not a symlink
		if fi, err := root.Lstat(ctx, path); err == nil && fi.Mode().IsRegular() {
			files[encoding] = fi
		}
	}

	offers := make([]string, 0, len(files)+1)
	for _, encoding := range compressedEncodingsPriority {
		if _, ok := files[encoding]; ok {
			offers = append(offers, encoding)
		}
	}
	offers = append(offers, "identity")

	acceptedEncoding := httputil.NegotiateContentEncoding(r, offers)

	if fi, ok := files[acceptedEncoding]; ok {
		w.Header().Set("Content-Encoding", acceptedEncoding)

		// http.ServeContent doesn't set Content-Length if Content-Encoding is set
		w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))

		return fullPath + compressedEncodings[acceptedEncoding]
	}

	return fullPath
}
