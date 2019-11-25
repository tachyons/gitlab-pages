package gitlab

import (
	"errors"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/cache"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

// Gitlab source represent a new domains configuration source. We fetch all the
// information about domains from GitLab instance.
type Gitlab struct {
	client *client.Client
	cache  *cache.Cache
}

// New returns a new instance of gitlab domain source.
func New(config client.Config) *Gitlab {
	return &Gitlab{client: client.NewFromConfig(config), cache: cache.New()}
}

// GetDomain return a representation of a domain that we have fetched from
// GitLab
func (g *Gitlab) GetDomain(name string) (*domain.Domain, error) {
	return nil, errors.New("not implemented")
}

// Resolve is supposed to get the serving lookup path based on the request from
// the GitLab source
func (g *Gitlab) Resolve(*http.Request) (*serving.LookupPath, string, error) {
	return nil, "", nil
}
