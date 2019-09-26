package domain

import "net/http"

// Resolver represents an interface responsible for resolving a project
// per-request
type Resolver interface {
	// Resolve returns a project with a file path and an error if it occured
	Resolve(*http.Request) (*Project, string, error)
}
