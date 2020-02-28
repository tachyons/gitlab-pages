package domain

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// Resolver represents an interface responsible for resolving a pages serving
// request for each HTTP request
type Resolver interface {
	// Resolve returns a serving request and an error if it occurred
	Resolve(*http.Request) (*serving.Request, error)
}
