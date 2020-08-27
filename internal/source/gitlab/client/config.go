package client

import "time"

// Config represents an interface that is configuration provider for client
// capable of comunicating with GitLab
type Config interface {
	RootDomain() string
	InternalGitLabServerURL() string
	GitlabAPISecret() []byte
	GitlabClientConnectionTimeout() time.Duration
	GitlabJWTTokenExpiry() time.Duration
	DomainConfigSource() string
}
