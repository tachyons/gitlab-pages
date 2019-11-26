package gitlab

import "gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/domain"

// Client interace represents a client capable of fetching a virtual domain
// from an external API
type Client interface {
	// GetVirtualDomain retrieves a virtual domain from an external API
	GetVirtualDomain(host string) (*domain.VirtualDomain, error)
}
