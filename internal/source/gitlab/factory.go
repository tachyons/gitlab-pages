package gitlab

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/file/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/file/zip"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/serverless"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// fabricateLookupPath fabricates a serving LookupPath based on the API LookupPath
// `size` argument is DEPRECATED, see
// https://gitlab.com/gitlab-org/gitlab-pages/issues/272
func fabricateLookupPath(size int, lookup api.LookupPath) *serving.LookupPath {
	return &serving.LookupPath{
		Prefix:             lookup.Prefix,
		Path:               strings.TrimPrefix(lookup.Source.Path, "/"),
		IsNamespaceProject: (lookup.Prefix == "/" && size > 1),
		IsHTTPSOnly:        lookup.HTTPSOnly,
		HasAccessControl:   lookup.AccessControl,
		ProjectID:          uint64(lookup.ProjectID),
	}
}

// fabricateServing fabricates serving based on the GitLab API response
func fabricateServing(lookup api.LookupPath) serving.Serving {
	source := lookup.Source

	switch source.Type {
	case "file":
		return disk.New()
	case "zip":
		return zip.New()
	case "serverless":
		serving, err := serverless.NewFromAPISource(source.Serverless)
		if err != nil {
			log.WithError(err).Errorf("could not fabricate serving for project %d", lookup.ProjectID)

			break
		}

		return serving
	}

	return defaultServing()
}

func defaultServing() serving.Serving {
	return disk.New()
}
