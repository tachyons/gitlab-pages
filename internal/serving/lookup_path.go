package serving

import (
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// LookupPath holds a domain project configuration needed to handle a request
type LookupPath struct {
	Prefix             string // Project prefix, for example, /my/project in group.gitlab.io/my/project/index.html
	Path               string // Path is an internal and serving-specific location of a document
	IsNamespaceProject bool   // IsNamespaceProject is DEPRECATED, see https://gitlab.com/gitlab-org/gitlab-pages/issues/272
	IsHTTPSOnly        bool
	HasAccessControl   bool
	ProjectID          uint64
}

// NewLookupPath fabricates a new serving lookup path based on a API response.
// `lookups` argument is a temporary workaround for
// https://gitlab.com/gitlab-org/gitlab-pages/issues/272
func NewLookupPath(lookups int, lookup api.LookupPath) *LookupPath {
	return &LookupPath{
		Prefix:             lookup.Prefix,
		Path:               strings.TrimPrefix(lookup.Source.Path, "/"),
		IsNamespaceProject: (lookup.Prefix == "/" && lookups > 1),
		IsHTTPSOnly:        lookup.HTTPSOnly,
		HasAccessControl:   lookup.AccessControl,
		ProjectID:          uint64(lookup.ProjectID),
	}
}
