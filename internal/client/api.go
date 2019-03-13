package client

// API implements simple interface that allows
// Pages to talk and request data from GitLab
type API interface {
	RequestDomain(host string) (*DomainResponse, error)
	IsReady() bool
}
