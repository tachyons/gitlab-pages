package disk

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// Serving describes a disk access serving
type Serving struct {
	Reader
}

// ServeFileHTTP serves a file from disk and returns true. It returns false
// when a file could not been found.
func (s *Serving) ServeFileHTTP(h serving.Handler) bool {
	if s.tryFile(h) == nil {
		return true
	}

	return false
}

// ServeNotFoundHTTP tries to read a custom 404 page
func (s *Serving) ServeNotFoundHTTP(h serving.Handler) {
	if s.tryNotFound(h) == nil {
		return
	}

	// Generic 404
	httperrors.Serve404(h.Writer)
}

// New returns a serving instance that is capable of reading files
// from the disk
func New() serving.Serving {
	return &Serving{}
}
