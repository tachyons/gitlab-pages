package factory

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/serverless"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// Serving fabricates serving based on the GitLab API response
func Serving(lookup api.LookupPath) serving.Serving {
	source := lookup.Source

	switch source.Type {
	case "file":
		return disk.New()
	case "serverless":
		serving, err := serverless.NewFromAPISource(source.Serverless)
		if err != nil {
			break
		}

		return serving
	}

	return DefaultServing()
}

// DefaultServing returns a serving that we will use as a default one, for
// example to show an error, if API response does not allow us to properly
// fabricate a serving
func DefaultServing() serving.Serving {
	return disk.New()
}
