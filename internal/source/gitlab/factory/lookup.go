package factory

import (
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// LookupPath fabricates a serving LookupPath based on the API LookupPath
// `size` argument is DEPRECATED, see
// https://gitlab.com/gitlab-org/gitlab-pages/issues/272
func LookupPath(size int, lookup api.LookupPath) *serving.LookupPath {
	return &serving.LookupPath{
		Prefix:             lookup.Prefix,
		Path:               strings.TrimPrefix(lookup.Source.Path, "/"),
		IsNamespaceProject: (lookup.Prefix == "/" && size > 1),
		IsHTTPSOnly:        lookup.HTTPSOnly,
		HasAccessControl:   lookup.AccessControl,
		ProjectID:          uint64(lookup.ProjectID),
	}
}
