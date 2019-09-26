package disk

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

type customProjectResolver struct {
	config *domainConfig
}

// TODO tests
func (p *customProjectResolver) Resolve(r *http.Request) (*domain.Project, string, error) {
	project := &domain.Project{
		LookupPath:         "/",
		IsNamespaceProject: false,
		IsHTTPSOnly:        p.config.HTTPSOnly,
		HasAccessControl:   p.config.AccessControl,
		ID:                 p.config.ID,
	}

	return project, r.URL.Path, nil
}
