package source

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
)

// Domains struct represents a map of all domains supported by pages. It is
// currently reading them from disk.
type Domains struct {
	Source
}

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func NewDomains() *Domains {
	return &Domains{
		Source: disk.New(),
	}
}
