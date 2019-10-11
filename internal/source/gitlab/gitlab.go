package gitlab

import "gitlab.com/gitlab-org/gitlab-pages/internal/domain"

// Gitlab source represent a new domains configuration source. We fetch all the
// information about domains from GitLab instance.
type Gitlab struct {
	client
	lookups []Lookup
}

// GetDomain return a representation of a domain that we have fetched from
// GitLab
func (g *Gitlab) GetDomain(name string) *domain.Domain {
	return nil
}

// HasDomain checks if a domain is known to GitLab
func (g *Gitlab) HasDomain(name string) bool {
	return false
}

// Watch starts Gitlab domains source
func (g *Gitlab) Watch(rootDomain string) {
}

// Ready checks if Gitlab domains source can be used
func (g *Gitlab) Ready() bool {
	return false
}
