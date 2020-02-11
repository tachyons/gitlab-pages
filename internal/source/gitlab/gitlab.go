package gitlab

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"
	"time"

	store "github.com/patrickmn/go-cache"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/cache"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

// Gitlab source represent a new domains configuration source. We fetch all the
// information about domains from GitLab instance.
type Gitlab struct {
	client api.Resolver
	store  *store.Cache
}

// New returns a new instance of gitlab domain source.
func New(config client.Config) (*Gitlab, error) {
	client, err := client.NewFromConfig(config)
	if err != nil {
		return nil, err
	}

	return &Gitlab{
		client: cache.NewCache(client),
		store:  store.New(time.Hour, time.Minute),
	}, nil
}

// GetDomain return a representation of a domain that we have fetched from
// GitLab
func (g *Gitlab) GetDomain(name string) (*domain.Domain, error) {
	lookup := g.client.Resolve(context.Background(), name)

	if lookup.Error != nil {
		return nil, lookup.Error
	}

	// Domain does not exist
	if lookup.Domain == nil {
		return nil, nil
	}

	domain := domain.Domain{
		Name:            name,
		CertificateCert: lookup.Domain.Certificate,
		CertificateKey:  lookup.Domain.Key,
		Resolver:        g,
	}

	return &domain, nil
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
	lookups := len(response.Domain.LookupPaths)

	for _, lookup := range response.Domain.LookupPaths {
		isSubPath := strings.HasPrefix(urlPath, lookup.Prefix)
		isRootPath := urlPath == path.Clean(lookup.Prefix)

		if isSubPath || isRootPath {
			lookupPath := &serving.LookupPath{
				Prefix:             lookup.Prefix,
				Path:               strings.TrimPrefix(lookup.Source.Path, "/"),
				IsNamespaceProject: (lookup.Prefix == "/" && lookups > 1),
				IsHTTPSOnly:        lookup.HTTPSOnly,
				HasAccessControl:   lookup.AccessControl,
				ProjectID:          uint64(lookup.ProjectID),
			}

			subPath := ""
			if isSubPath {
				subPath = strings.TrimPrefix(urlPath, lookup.Prefix)
			}

			return &serving.Request{
				Serving:    disk.New(),
				LookupPath: lookupPath,
				SubPath:    subPath}, nil
		}
	}

	return &serving.Request{Serving: disk.New()}, errors.New("could not match lookup path")
}
