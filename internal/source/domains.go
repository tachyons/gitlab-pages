package source

import (
	"context"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
)

// Domains struct represents a map of all domains supported by pages.
type Domains struct {
	gitlab Source
}

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func NewDomains(cfg *config.GitLab) (*Domains, error) {
	domains := &Domains{}
	if err := domains.setConfigSource(cfg); err != nil {
		return nil, err
	}

	return domains, nil
}

// setConfigSource and initialize gitlab source
func (d *Domains) setConfigSource(cfg *config.GitLab) error {
	return d.setGitLabClient(cfg)
}

func (d *Domains) setGitLabClient(cfg *config.GitLab) error {
	// We want to notify users about any API issues
	// Creating a glClient will start polling connectivity in the background
	// and spam errors in log
	glClient, err := gitlab.New(cfg)
	if err != nil {
		return err
	}

	d.gitlab = glClient

	return nil
}

// GetDomain retrieves a domain information from a source. We are using two
// sources here because it allows us to switch behavior and the domain source
// for some subset of domains, to test / PoC the new GitLab Domains Source that
// we plan to use to replace the disk source.
func (d *Domains) GetDomain(ctx context.Context, name string) (*domain.Domain, error) {
	return d.gitlab.GetDomain(ctx, name)
}
