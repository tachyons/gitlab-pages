package gitlab

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"

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
	client api.Resolver
}

// New returns a new instance of gitlab domain source.
func New(config client.Config) (*Gitlab, error) {
	client, err := client.NewFromConfig(config)
	if err != nil {
		return nil, err
	}

	return &Gitlab{client: cache.NewCache(client)}, nil
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

// Resolve is supposed to get the serving lookup path based on the request from
// the GitLab source
func (g *Gitlab) Resolve(r *http.Request) (*serving.LookupPath, string, error) {
	host := request.GetHostWithoutPort(r)

	response := g.client.Resolve(r.Context(), host)
	if response.Error != nil {
		return nil, "", response.Error
	}

	urlPath := path.Clean(r.URL.Path)

	for _, lookup := range response.Domain.LookupPaths {
		isSubPath := strings.HasPrefix(urlPath, lookup.Prefix)
		isRootPath := urlPath == path.Clean(lookup.Prefix)

		if isSubPath || isRootPath {
			lookupPath := &serving.LookupPath{
				Prefix:             lookup.Prefix,
				Path:               strings.TrimPrefix(lookup.Source.Path, "/"),
				IsNamespaceProject: (lookup.Prefix == "/" && len(response.Domain.LookupPaths) > 1),
				IsHTTPSOnly:        lookup.HTTPSOnly,
				HasAccessControl:   lookup.AccessControl,
				ProjectID:          uint64(lookup.ProjectID),
			}

			subPath := ""
			if isSubPath {
				subPath = strings.TrimPrefix(urlPath, lookup.Prefix)
			}

			return lookupPath, subPath, nil
		}
	}

	return nil, "", errors.New("could not match lookup path")
}
