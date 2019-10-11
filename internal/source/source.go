package source

import "gitlab.com/gitlab-org/gitlab-pages/internal/domain"

type Source interface {
	GetDomain(string) *domain.Domain
	HasDomain(string) bool
	Watch(rootDomain string)
	Ready() bool
}
