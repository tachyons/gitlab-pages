package disk

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

// Custom serving represent a resource that can be served from a directory
// representing GitLab project
type Project struct {
	Location string
	*Reader
}

// ServeFileHTTP returns true if something was served, false if not.
func (p *Project) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	return p.serveFileFromConfig(w, r)
}

// ServeNotFoundHTTP serves the not found pages from the projects.
func (p *Project) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	p.serveNotFoundFromConfig(w, r)
}

func (p *Project) HasAcmeChallenge(token string) bool {
	_, err := p.resolvePath(p.Location, ".well-known/acme-challenge", token)
	// there is an acme challenge on disk
	if err == nil {
		return true
	}

	_, err = p.resolvePath(p.Location, ".well-known/acme-challenge", token, "index.html")
	if err == nil {
		return true
	}

	return false
}

func (p *Project) serveFileFromConfig(w http.ResponseWriter, r *http.Request) bool {
	// Try to serve file for http://host/... => /group/project/...
	if p.tryFile(w, r, p.Location, r.URL.Path) == nil {
		return true
	}

	return false
}

func (p *Project) serveNotFoundFromConfig(w http.ResponseWriter, r *http.Request) {
	// Try serving not found page for http://host/ => /group/project/404.html
	if p.tryNotFound(w, r, p.Location) == nil {
		return
	}

	// Serve generic not found
	httperrors.Serve404(w)
}
