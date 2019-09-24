package disk

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

// Group serving represents a resource that can be served from a directory
// representing GitLab group
type Group struct {
	Resolver
	*Reader
}

type Resolver interface {
	ProjectWithSubpath(*http.Request) (string, string, error)
}

// ServeFileHTTP returns true if something was served, false if not.
func (g *Group) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	return g.serveFileFromGroup(w, r)
}

// ServeNotFoundHTTP serves the not found pages from the projects.
func (g *Group) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	g.serveNotFoundFromGroup(w, r)
}

func (g *Group) HasAcmeChallenge(token string) bool {
	return false
}

func (g *Group) serveFileFromGroup(w http.ResponseWriter, r *http.Request) bool {
	projectName, subPath, err := g.Resolver.ProjectWithSubpath(r)

	if err != nil {
		httperrors.Serve404(w)
		return true
	}

	if g.tryFile(w, r, projectName, subPath) == nil {
		return true
	}

	return false
}

func (g *Group) serveNotFoundFromGroup(w http.ResponseWriter, r *http.Request) {
	projectName, _, err := g.Resolver.ProjectWithSubpath(r)

	if err != nil {
		httperrors.Serve404(w)
		return
	}

	// Try serving custom not-found page
	if g.tryNotFound(w, r, projectName) == nil {
		return
	}

	// Generic 404
	httperrors.Serve404(w)
}
