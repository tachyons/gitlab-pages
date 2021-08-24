package gitlab

import (
	"context"
	"errors"
	"net/http"
	"path"
	"sort"
	"strings"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
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
	client     api.Resolver
	enableDisk bool
}

// New returns a new instance of gitlab domain source.
func New(cfg *config.GitLab) (*Gitlab, error) {
	glClient, err := client.NewFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	g := &Gitlab{
		client:     cache.NewCache(glClient, &cfg.Cache),
		enableDisk: cfg.EnableDisk,
	}

	return g, nil
}

// GetDomain return a representation of a domain that we have fetched from
// GitLab
func (g *Gitlab) GetDomain(ctx context.Context, name string) (*domain.Domain, error) {
	lookup := g.client.Resolve(ctx, name)

	if lookup.Error != nil {
		if errors.Is(lookup.Error, client.ErrUnauthorizedAPI) {
			log.WithError(lookup.Error).Error("Pages cannot communicate with an instance of the GitLab API. Please sync your gitlab-secrets.json file: https://docs.gitlab.com/ee/administration/pages/#pages-cannot-communicate-with-an-instance-of-the-gitlab-api")
		}

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

	sortLookupsByPrefixLengthDesc(response.Domain.LookupPaths)

	for _, lookup := range response.Domain.LookupPaths {
		isSubPath := strings.HasPrefix(urlPath, lookup.Prefix)
		isRootPath := urlPath == path.Clean(lookup.Prefix)

		if isSubPath || isRootPath {
			subPath := ""
			if isSubPath {
				subPath = strings.TrimPrefix(urlPath, lookup.Prefix)
			}

			srv, err := g.fabricateServing(lookup)
			if err != nil {
				return nil, err
			}

			return &serving.Request{
				Serving:    srv,
				LookupPath: fabricateLookupPath(size, lookup),
				SubPath:    subPath}, nil
		}
	}
	logging.LogRequest(r).WithError(domain.ErrDomainDoesNotExist).Errorf("no lookup path for: %q", r.URL.Path)
	return nil, domain.ErrDomainDoesNotExist
}

// Ensure lookupPaths are sorted by prefix length to ensure the group level
// domain with prefix "/" is the last one to be checked.
// See https://gitlab.com/gitlab-org/gitlab-pages/-/issues/576
func sortLookupsByPrefixLengthDesc(lookups []api.LookupPath) {
	sort.SliceStable(lookups, func(i, j int) bool {
		return len(lookups[i].Prefix) > len(lookups[j].Prefix)
	})
}

// IsReady returns the value of Gitlab `isReady` which is updated by `Poll`.
func (g *Gitlab) IsReady() bool {
	return true
}
