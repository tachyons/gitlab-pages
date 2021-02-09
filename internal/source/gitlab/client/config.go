package client

import (
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
)

// Config represents an interface that is configuration provider for client
// capable of comunicating with GitLab
type Config interface {
	InternalGitLabServerURL() string
	GitlabAPISecret() []byte
	GitlabClientConnectionTimeout() time.Duration
	GitlabJWTTokenExpiry() time.Duration
	DomainConfigSource() string
	Cache() *config.Cache
}
