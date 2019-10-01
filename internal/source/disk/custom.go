package disk

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

type customProjectResolver struct {
	config *domainConfig

	path string
}

// TODO tests
func (p *customProjectResolver) Resolve(r *http.Request) (*serving.LookupPath, string, error) {
	project := &serving.LookupPath{
		Location:           "/",
		Path:               p.path,
		IsNamespaceProject: false,
		IsHTTPSOnly:        p.config.HTTPSOnly,
		HasAccessControl:   p.config.AccessControl,
		ID:                 p.config.ID,
	}

	return project, r.URL.Path, nil
}
