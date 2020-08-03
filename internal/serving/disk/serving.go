package disk

import (
	"log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Disk describes a disk access serving
type Disk struct {
	reader Reader
}

// ServeFileHTTP serves a file from disk and returns true. It returns false
// when a file could not been found.
func (s *Disk) ServeFileHTTP(h serving.Handler) bool {
	if err := s.reader.tryFile(h); err != nil {
		log.Print(err)
		return false
	}

	return true
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
// from the disk
func New() serving.Serving {
	return &Disk{
		reader: Reader{
			fileSizeMetric: metrics.DiskServingFileSize,
		},
	}
}
