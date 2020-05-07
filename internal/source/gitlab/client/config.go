package client

import "time"

// Config represents an interface that is configuration provider for client
// capable of comunicating with GitLab
type Config interface {
	InternalGitLabServerURL() string
	GitlabAPISecret() []byte
	GitlabClientConnectionTimeout() time.Duration
	GitlabJWTTokenExpiry() time.Duration
}
