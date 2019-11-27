package source

import (
	"errors"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
)

var newSourceDomains = []string{
	"pages-project-poc.gitlab.io",
	"pages-namespace-poc.gitlab.io",
	"pages-custom-poc.grzegorz.co",
	"new-source.test.io", // used in acceptance tests
}

var brokenSourceDomain = "pages-broken-poc.gitlab.io"

// Domains struct represents a map of all domains supported by pages. It is
// currently using two sources during the transition to the new GitLab domains
// source.
type Domains struct {
	gitlab Source
	disk   *disk.Disk // legacy disk source
}

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func NewDomains(config Config) *Domains {
	return &Domains{
		disk:   disk.New(),
		gitlab: gitlab.New(config),
	}
}

// GetDomain retrieves a domain information from a source. We are using two
// sources here because it allows us to switch behavior and the domain source
// for some subset of domains, to test / PoC the new GitLab Domains Source that
// we plan to use to replace the disk source.
func (d *Domains) GetDomain(name string) (*domain.Domain, error) {
	if name == brokenSourceDomain {
		return nil, errors.New("broken test domain used")
	}

	return d.source(name).GetDomain(name)
}

// Start starts the disk domain source. It is DEPRECATED, because we want to
// remove it entirely when disk source gets removed.
func (d *Domains) Read(rootDomain string) {
	d.disk.Read(rootDomain)
}

// IsReady checks if the disk domain source managed to traverse entire pages
// filesystem and is ready for use. It is DEPRECATED, because we want to remove
// it entirely when disk source gets removed.
func (d *Domains) IsReady() bool {
	return d.disk.IsReady()
}

func (d *Domains) source(domain string) Source {
	for _, name := range newSourceDomains {
		if domain == name {
			return d.gitlab
		}
	}

	return d.disk
}
