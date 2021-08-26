package source

import (
	"context"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

// Source represents an abstract interface of a domains configuration source.
type Source interface {
	GetDomain(context.Context, string) (*domain.Domain, error)
}
