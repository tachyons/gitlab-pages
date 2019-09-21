package source

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
)

// Source represents a source of information about a domain. Whenever a request
// appears a concret implementation of a Source should find a domain that is
// needed to handle the request and serve pages
type Source interface {
	GetDomain(*http.Request) *domain.Domain
}
