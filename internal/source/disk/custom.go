package disk

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

type customProjectResolver struct {
	config *domainConfig

	path string
}

func (p *customProjectResolver) Resolve(r *http.Request) (*serving.LookupPath, string, error) {
	lookupPath := &serving.LookupPath{
		Prefix:             "/",
		Path:               p.path,
		IsNamespaceProject: false,
		IsHTTPSOnly:        p.config.HTTPSOnly,
		HasAccessControl:   p.config.AccessControl,
		ProjectID:          p.config.ID,
	}

	return lookupPath, r.URL.Path, nil
}
