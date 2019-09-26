package serving

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
)

// Serving is an interface used to define a serving driver
type Serving interface {
	ServeFileHTTP(Handler) bool
	ServeNotFoundHTTP(Handler)
	HasAcmeChallenge(handler Handler, token string) bool
}

// NewDiskServing returns a serving instance that is capable of reading files
// from the disk
func NewDiskServing(domain, location string) Serving {
	return &diskServing{
		disk: &disk.Serving{
			Domain: domain,
			Reader: &disk.Reader{Location: location},
		},
	}
}
