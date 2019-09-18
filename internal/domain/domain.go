package domain

import (
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/host"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

const (
	subgroupScanLimit int = 21
	// maxProjectDepth is set to the maximum nested project depth in gitlab (21) plus 3.
	// One for the project, one for the first empty element of the split (URL.Path starts with /),
	// and one for the real file path
	maxProjectDepth int = subgroupScanLimit + 3
)

// Project represents a projects associated with a domain / requested domain
// path
type project struct {
	NamespaceProject bool
	HTTPSOnly        bool
	AccessControl    bool
	ID               uint64
}

// Domain is a domain that gitlab-pages can serve.
type Domain struct {
	source *source.Disk

	// custom domains
	project string
	group   group
	config  *Config

	certificate      *tls.Certificate
	certificateError error
	certificateOnce  sync.Once
}

// String implements Stringer.
func (d *Domain) String() string {
	if d.group.name != "" && d.project != "" {
		return d.group.name + "/" + d.project
	}

	if d.group.name != "" {
		return d.group.name
	}

	return d.project
}

// Look up a project inside the domain based on the host and path. Returns the
// project and its name (if applicable)
func (d *Domain) getProjectWithSubpath(r *http.Request) (*project, string, string) {
	// Check for a project specified in the URL: http://group.gitlab.io/projectA
	// If present, these projects shadow the group domain.
	split := strings.SplitN(r.URL.Path, "/", maxProjectDepth)
	if len(split) >= 2 {
		project, projectPath, urlPath := d.group.digProjectWithSubpath("", split[1:])
		if project != nil {
			return project, projectPath, urlPath
		}
	}

	// Since the URL doesn't specify a project (e.g. http://mydomain.gitlab.io),
	// return the group project if it exists.
	if host := host.FromRequest(r); host != "" {
		if groupProject := d.group.projects[host]; groupProject != nil {
			return groupProject, host, strings.Join(split[1:], "/")
		}
	}

	return nil, "", ""
}

// IsHTTPSOnly figures out if the request should be handled with HTTPS
// only by looking at group and project level config.
func (d *Domain) IsHTTPSOnly(r *http.Request) bool {
	if d == nil {
		return false
	}

	// Check custom domain config (e.g. http://example.com)
	if d.config != nil {
		return d.config.HTTPSOnly
	}

	// Check projects served under the group domain, including the default one
	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return project.HTTPSOnly
	}

	return false
}

// IsAccessControlEnabled figures out if the request is to a project that has access control enabled
func (d *Domain) IsAccessControlEnabled(r *http.Request) bool {
	if d == nil {
		return false
	}

	// Check custom domain config (e.g. http://example.com)
	if d.config != nil {
		return d.config.AccessControl
	}

	// Check projects served under the group domain, including the default one
	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return project.AccessControl
	}

	return false
}

// HasAcmeChallenge checks domain directory contains particular acme challenge
func (d *Domain) HasAcmeChallenge(token string) bool {
	if d == nil {
		return false
	}

	return d.source.HasAcmeChallenge(token)
}

// IsNamespaceProject figures out if the request is to a namespace project
func (d *Domain) IsNamespaceProject(r *http.Request) bool {
	if d == nil {
		return false
	}

	// If request is to a custom domain, we do not handle it as a namespace project
	// as there can't be multiple projects under the same custom domain
	if d.config != nil {
		return false
	}

	// Check projects served under the group domain, including the default one
	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return project.NamespaceProject
	}

	return false
}

// GetID figures out what is the ID of the project user tries to access
func (d *Domain) GetID(r *http.Request) uint64 {
	if d == nil {
		return 0
	}

	if d.config != nil {
		return d.config.ID
	}

	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return project.ID
	}

	return 0
}

// HasProject figures out if the project exists that the user tries to access
func (d *Domain) HasProject(r *http.Request) bool {
	if d == nil {
		return false
	}

	if d.config != nil {
		return true
	}

	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return true
	}

	return false
}

// EnsureCertificate parses the PEM-encoded certificate for the domain
func (d *Domain) EnsureCertificate() (*tls.Certificate, error) {
	if d.config == nil {
		return nil, errors.New("tls certificates can be loaded only for pages with configuration")
	}

	d.certificateOnce.Do(func() {
		var cert tls.Certificate
		cert, d.certificateError = tls.X509KeyPair([]byte(d.config.Certificate), []byte(d.config.Key))
		if d.certificateError == nil {
			d.certificate = &cert
		}
	})

	return d.certificate, d.certificateError
}

func (d *Domain) getServing(w http.ResponseWriter, r *http.Request) *source.Serving {
	_, project, subPath := d.getProjectWithSubpath(r)

	return &source.Serving{
		Writer: w, Request: r, Project: project, SubPath: subPath,
	}
}

func (d *Domain) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	if !d.IsAccessControlEnabled(r) {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	serving := d.getServing(w, r)

	return d.source.ServeFileHTTP(serving)
}

func (d *Domain) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	serving := d.getServing(w, r)

	d.source.ServeNotFoundHTTP(serving)
}
