package disk

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
)

type customProjectResolver struct {
	config *domainConfig

	path string
}

func (p *customProjectResolver) Resolve(r *http.Request) (*serving.Request, error) {
	lookupPath := &serving.LookupPath{
		Prefix:             "/",
		Path:               p.path,
		IsNamespaceProject: false,
		IsHTTPSOnly:        p.config.HTTPSOnly,
		HasAccessControl:   p.config.AccessControl,
		ProjectID:          p.config.ID,
	}

	return &serving.Request{
		Serving:    disk.Instance(),
		LookupPath: lookupPath,
		SubPath:    r.URL.Path,
	}, nil
}
