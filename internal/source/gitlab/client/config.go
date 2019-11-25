package client

// Config represents an interface that is configuration provider for client
// capable of comunicating with GitLab
type Config interface {
	GitlabServerURL() string
	GitlabClientSecret() []byte
}
