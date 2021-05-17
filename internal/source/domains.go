package source

import (
	"regexp"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
)

var (
	// serverlessDomainRegex is a regular expression we use to check if a domain
	// is a serverless domain, to short circuit gitlab source rollout. It can be
	// removed after the rollout is done
	// TODO: remove serverless
	serverlessDomainRegex = regexp.MustCompile(`^[^.]+-[[:xdigit:]]{2}a1[[:xdigit:]]{10}f2[[:xdigit:]]{2}[[:xdigit:]]+-?.*`)
)

// Domains struct represents a map of all domains supported by pages. It is
// currently using two sources during the transition to the new GitLab domains
// source.
type Domains struct {
	gitlab Source
}

// NewDomains is a factory method for domains initializing a mutex. It should
// not initialize `dm` as we later check the readiness by comparing it with a
// nil value.
func NewDomains(cfg *config.GitLab) (*Domains, error) {
	glClient, err := gitlab.New(cfg)
	if err != nil {
		return nil, err
	}

	return &Domains{
		gitlab: glClient,
	}, nil
}

// GetDomain retrieves a domain information from a source. We are using two
// sources here because it allows us to switch behavior and the domain source
// for some subset of domains, to test / PoC the new GitLab Domains Source that
// we plan to use to replace the disk source.
func (d *Domains) GetDomain(name string) (*domain.Domain, error) {
	return d.gitlab.GetDomain(name)
}

// IsReady checks if the disk domain source managed to traverse entire pages
// filesystem and is ready for use. It is DEPRECATED, because we want to remove
// it entirely when disk source gets removed.
func (d *Domains) IsReady() bool {
	return d.gitlab.IsReady()
}

// IsServerlessDomain checks if a domain requested is a serverless domain we
// need to handle differently.
//
// Domain is a serverless domain when it matches `serverlessDomainRegex`. The
// regular expression is also defined on the gitlab-rails side, see
// https://gitlab.com/gitlab-org/gitlab/-/blob/master/app/models/serverless/domain.rb#L7
// TODO: remove serverless
func IsServerlessDomain(domain string) bool {
	return serverlessDomainRegex.MatchString(domain)
}
