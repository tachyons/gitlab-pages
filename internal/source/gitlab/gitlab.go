package gitlab

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/cache"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

// Gitlab source represent a new domains configuration source. We fetch all the
// information about domains from GitLab instance.
type Gitlab struct {
	client api.Client
	cache  *cache.Cache // WIP
}

// New returns a new instance of gitlab domain source.
func New(config client.Config) *Gitlab {
	return &Gitlab{client: client.NewFromConfig(config), cache: cache.New()}
}

// GetDomain return a representation of a domain that we have fetched from
// GitLab
func (g *Gitlab) GetDomain(name string) (*domain.Domain, error) {
	lookup := g.client.GetLookup(context.Background(), name)

	// NoContent response means that a domain does not exist
	if lookup.Status == http.StatusNoContent {
		return nil, nil
	}

	if lookup.Error != nil {
		return nil, lookup.Error
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
	response := g.client.GetLookup(r.Context(), r.Host)
	if response.Error != nil {
		return nil, "", response.Error
	}

	for _, lookup := range response.Domain.LookupPaths {
		urlPath := path.Clean(r.URL.Path)

		if strings.HasPrefix(urlPath, lookup.Prefix) {
			lookupPath := &serving.LookupPath{
				Prefix:             lookup.Prefix,
				Path:               strings.TrimPrefix(lookup.Source.Path, "/"),
				IsNamespaceProject: (lookup.Prefix == "/" && len(response.Domain.LookupPaths) > 1),
				IsHTTPSOnly:        lookup.HTTPSOnly,
				HasAccessControl:   lookup.AccessControl,
				ProjectID:          uint64(lookup.ProjectID),
			}

			requestPath := strings.TrimPrefix(urlPath, lookup.Prefix)

			return lookupPath, strings.TrimPrefix(requestPath, "/"), nil
		}
	}

	return nil, "", errors.New("could not match lookup path")
}
