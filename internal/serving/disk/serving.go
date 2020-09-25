package disk

import (
	"mime"
	"os"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Disk describes a disk access serving
type Disk struct {
	reader Reader
}

// ServeFileHTTP serves a file from disk and returns true. It returns false
// when a file could not been found.
func (s *Disk) ServeFileHTTP(h serving.Handler) bool {
	if s.reader.tryFile(h) == nil {
		return true
	}

	if os.Getenv("FF_ENABLE_REDIRECTS") != "false" {
		if s.reader.tryRedirects(h) == nil {
			return true
		}
	}

	return false
}

// ServeNotFoundHTTP tries to read a custom 404 page
func (s *Disk) ServeNotFoundHTTP(h serving.Handler) {
	if s.reader.tryNotFound(h) == nil {
		return
	}

	// Generic 404
	httperrors.Serve404(h.Writer)
}

// New returns a serving instance that is capable of reading files
// from the VFS
func New(vfs vfs.VFS) serving.Serving {
	addExtraMIMETypes(extraMIMETypes)

	return &Disk{
		reader: Reader{
			fileSizeMetric: metrics.DiskServingFileSize,
			vfs:            vfs,
		},
	}
}

var extraMIMETypes = map[string]string{
	".avif": "image/avif",
}

func addExtraMIMETypes(mimeTypes map[string]string) {
	for ext, mimeType := range mimeTypes {
		if err := mime.AddExtensionType(ext, mimeType); err != nil {
			log.WithError(err).Errorf("failed to add extension: %q with MIME type: %q", ext, mimeType)
		}
	}
}
