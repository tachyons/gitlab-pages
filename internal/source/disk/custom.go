package disk

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/local"
)

type customProjectResolver struct {
	config *domainConfig

	path string
}

func (p *customProjectResolver) Resolve(r *http.Request) (*serving.Request, error) {
	lookupPath := &serving.LookupPath{
		ServingType:        "file",
		Prefix:             "/",
		Path:               p.path,
		IsNamespaceProject: false,
		IsHTTPSOnly:        p.config.HTTPSOnly,
		HasAccessControl:   p.config.AccessControl,
		ProjectID:          p.config.ID,
		SubPath:            r.URL.Path,
	}

	return &serving.Request{
		Serving:    local.Instance(),
		LookupPath: lookupPath,
	}, nil
}
