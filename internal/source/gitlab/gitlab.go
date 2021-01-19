package gitlab

import (
	"context"
	"net/http"
	"path"
	"strings"
	"sync"

	"github.com/cenkalti/backoff/v4"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/cache"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

// Gitlab source represent a new domains configuration source. We fetch all the
// information about domains from GitLab instance.
type Gitlab struct {
	client  api.Resolver
	mu      *sync.RWMutex
	isReady bool
}

// New returns a new instance of gitlab domain source.
func New(config client.Config) (*Gitlab, error) {
	client, err := client.NewFromConfig(config)
	if err != nil {
		return nil, err
	}

	g := &Gitlab{
		client: cache.NewCache(client, nil),
		mu:     &sync.RWMutex{},
	}

	go g.poll(backoff.DefaultInitialInterval, maxPollingTime)

	// using nil for cache config will use the default values specified in internal/source/gitlab/cache/cache.go#12
	return g, nil
}

// GetDomain return a representation of a domain that we have fetched from
// GitLab
func (g *Gitlab) GetDomain(name string) (*domain.Domain, error) {
	lookup := g.client.Resolve(context.Background(), name)

	if lookup.Error != nil {
		return nil, lookup.Error
	}

	// TODO introduce a second-level cache for domains, invalidate using etags
	// from first-level cache
	d := domain.New(name, lookup.Domain.Certificate, lookup.Domain.Key, g)

	return d, nil
}

// Resolve is supposed to return the serving request containing lookup path,
// subpath for a given lookup and the serving itself created based on a request
// from GitLab pages domains source
func (g *Gitlab) Resolve(r *http.Request) (*serving.Request, error) {
	host := request.GetHostWithoutPort(r)

	response := g.client.Resolve(r.Context(), host)
	if response.Error != nil {
		return nil, response.Error
	}

	urlPath := path.Clean(r.URL.Path)
	size := len(response.Domain.LookupPaths)

	for _, lookup := range response.Domain.LookupPaths {
		isSubPath := strings.HasPrefix(urlPath, lookup.Prefix)
		isRootPath := urlPath == path.Clean(lookup.Prefix)

		if isSubPath || isRootPath {
			subPath := ""
			if isSubPath {
				subPath = strings.TrimPrefix(urlPath, lookup.Prefix)
			}

			return &serving.Request{
				Serving:    fabricateServing(lookup),
				LookupPath: fabricateLookupPath(size, lookup),
				SubPath:    subPath}, nil
		}
	}

	return nil, domain.ErrDomainDoesNotExist
}

// IsReady returns the value of Gitlab `isReady` which is updated by `Poll`.
func (g *Gitlab) IsReady() bool {
	g.mu.RLock()
	defer g.mu.RUnlock()

	return g.isReady
}
