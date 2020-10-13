package disk

import (
	"net/http"
	"os"

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
func (s *Disk) ServeFileHTTP(w http.ResponseWriter, r *http.Request, lookupPath *serving.LookupPath) bool {
	if s.reader.tryFile(w, r, lookupPath) == nil {
		return true
	}

	if os.Getenv("FF_ENABLE_REDIRECTS") != "false" {
		if s.reader.tryRedirects(w, r, lookupPath) == nil {
			return true
		}
	}

	return false
}

// ServeNotFoundHTTP tries to read a custom 404 page
func (s *Disk) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request, lookupPath *serving.LookupPath) {
	if s.reader.tryNotFound(w, r, lookupPath) == nil {
		return
	}

	// Generic 404
	httperrors.Serve404(w)
}

// New returns a serving instance that is capable of reading files
// from the VFS
func New(vfs vfs.VFS) serving.Serving {
	return &Disk{
		reader: Reader{
			fileSizeMetric: metrics.DiskServingFileSize,
			vfs:            vfs,
		},
	}
}
