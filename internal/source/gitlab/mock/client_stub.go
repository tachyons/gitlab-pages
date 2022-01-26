package mock

import "gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"

type ClientStub interface {
	api.Client
	api.Resolver
}
