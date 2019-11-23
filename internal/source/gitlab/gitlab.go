package gitlab

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// Gitlab source represent a new domains configuration source. We fetch all the
// information about domains from GitLab instance.
type Gitlab struct {
	client Client
	cache  Cache
}

// New returns a new instance of gitlab domain source.
func New() *Gitlab {
	return &Gitlab{}
}

// GetDomain return a representation of a domain that we have fetched from
// GitLab
// It should return source.Lookup TODO
func (g *Gitlab) GetDomain(name string) *domain.Domain {
	return nil
}

// HasDomain checks if a domain is known to GitLab
// TODO lookup status code etc.
func (g *Gitlab) HasDomain(name string) bool {
	return g.GetDomain(name) != nil
}

// Resolve is supposed to get the serving lookup path based on the request from
// the GitLab source
func (g *Gitlab) Resolve(*http.Request) (*serving.LookupPath, string, error) {
	return nil, "", nil
}
