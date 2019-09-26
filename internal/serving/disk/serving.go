package disk

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

// Serving describes a disk access serving
type Serving struct {
	Domain string // TODO it is not used but might be handy
	*Reader
}

// ServeFileHTTP serves a file from disk and returns true. It returns false
// when a file could not been found.
func (s *Serving) ServeFileHTTP(h handler) bool {
	if s.tryFile(h) == nil {
		return true
	}

	return false
}

// ServeNotFoundHTTP tries to read a custom 404 page
func (s *Serving) ServeNotFoundHTTP(h handler) {
	if s.tryNotFound(h) == nil {
		return
	}

	// Generic 404
	httperrors.Serve404(h.Writer())
}

// HasAcmeChallenge checks if the ACME challenge is present on the disk
func (s *Serving) HasAcmeChallenge(h handler, token string) bool {
	_, err := s.resolvePath(h.LookupPath(), ".well-known/acme-challenge", token)
	// there is an acme challenge on disk
	if err == nil {
		return true
	}

	_, err = s.resolvePath(h.LookupPath(), ".well-known/acme-challenge", token, "index.html")
	if err == nil {
		return true
	}

	return false
}
