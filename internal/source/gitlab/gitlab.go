package gitlab

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/cache"
)

// Gitlab source represent a new domains configuration source. We fetch all the
// information about domains from GitLab instance.
type Gitlab struct {
	client Client
	cache  cache.Cache
}

// GetDomain return a representation of a domain that we have fetched from
// GitLab
func (g *Gitlab) GetDomain(name string) *domain.Domain {
	return nil
}

// HasDomain checks if a domain is known to GitLab
func (g *Gitlab) HasDomain(name string) bool {
	return g.GetDomain(name) != nil
}

// Resolve is supposed to get the serving lookup path based on the request from
// the GitLab source
func (g *Gitlab) Resolve(*http.Request) (*serving.LookupPath, string, error) {
	return nil, "", nil
}

// Watch starts Gitlab domains source TODO remove
func (g *Gitlab) Watch(rootDomain string) {
}

// Ready checks if Gitlab domains source can be used TODO remove
func (g *Gitlab) Ready() bool {
	return false
}
