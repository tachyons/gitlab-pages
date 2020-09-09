package disk

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
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

	if os.Getenv("FF_ENABLE_REDIRECTS") == "true" {
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
func New(vfs vfs.VFS, fileSizeMetric prometheus.Histogram) serving.Serving {
	return &Disk{
		reader: Reader{
			fileSizeMetric: fileSizeMetric,
			vfs:            vfs,
		},
	}
}
