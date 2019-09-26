package serving

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
)

// Serving is an interface used to define a serving driver
type Serving interface {
	ServeFileHTTP(http.ResponseWriter, *http.Request) bool
	ServeNotFoundHTTP(http.ResponseWriter, *http.Request)
	HasAcmeChallenge(token string) bool
}

func NewProjectDiskServing(project, group string) Serving {
	return &disk.Project{
		Location: project,
		Reader: &disk.Reader{
			Group: group,
		},
	}
}

func NewGroupDiskServing(group string, resolver disk.Resolver) Serving {
	return &disk.Group{
		Resolver: resolver,
		Reader: &disk.Reader{
			Group: group,
		},
	}
}
