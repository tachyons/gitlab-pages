package domain

import (
	"crypto/tls"
	"errors"
	"net/http"
	"sync"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// GroupConfig represents a per-request config for a group domain
type GroupConfig interface {
	IsHTTPSOnly(*http.Request) bool
	HasAccessControl(*http.Request) bool
	IsNamespaceProject(*http.Request) bool
	ProjectID(*http.Request) uint64
	ProjectExists(*http.Request) bool
}

// Domain is a domain that gitlab-pages can serve.
type Domain struct {
	Group   string
	Project string

	DomainName    string
	Certificate   string
	Key           string
	HTTPSOnly     bool
	ProjectID     uint64
	AccessControl bool

	GroupConfig GroupConfig // handles group domain config
	Serving     serving.Serving

	certificate      *tls.Certificate
	certificateError error
	certificateOnce  sync.Once
}

// String implements Stringer.
func (d *Domain) String() string {
	if d.Group != "" && d.Project != "" {
		return d.Group + "/" + d.Project
	}

	if d.Group != "" {
		return d.Group
	}

	return d.Project
}

func (d *Domain) isCustomDomain() bool {
	return d.GroupConfig == nil
}

// IsHTTPSOnly figures out if the request should be handled with HTTPS
// only by looking at group and project level config.
func (d *Domain) IsHTTPSOnly(r *http.Request) bool {
	if d == nil {
		return false
	}

	// Check custom domain config (e.g. http://example.com)
	if d.isCustomDomain() {
		return d.HTTPSOnly
	}

	// Check projects served under the group domain, including the default one
	return d.GroupConfig.IsHTTPSOnly(r)
}

// IsAccessControlEnabled figures out if the request is to a project that has access control enabled
func (d *Domain) IsAccessControlEnabled(r *http.Request) bool {
	if d == nil {
		return false
	}

	// Check custom domain config (e.g. http://example.com)
	if d.isCustomDomain() {
		return d.AccessControl
	}

	// Check projects served under the group domain, including the default one
	return d.GroupConfig.HasAccessControl(r)
}

// HasAcmeChallenge checks domain directory contains particular acme challenge
func (d *Domain) HasAcmeChallenge(token string) bool {
	if d == nil {
		return false
	}

	if !d.isCustomDomain() {
		return false
	}

	return d.Serving.HasAcmeChallenge(token)
}

// IsNamespaceProject figures out if the request is to a namespace project
func (d *Domain) IsNamespaceProject(r *http.Request) bool {
	if d == nil {
		return false
	}

	// If request is to a custom domain, we do not handle it as a namespace project
	// as there can't be multiple projects under the same custom domain
	if d.isCustomDomain() {
		return false
	}

	// Check projects served under the group domain, including the default one
	return d.GroupConfig.IsNamespaceProject(r)
}

// GetID figures out what is the ID of the project user tries to access
func (d *Domain) GetID(r *http.Request) uint64 {
	if d == nil {
		return 0
	}

	if d.isCustomDomain() {
		return d.ProjectID
	}

	return d.GroupConfig.ProjectID(r)
}

// HasProject figures out if the project exists that the user tries to access
func (d *Domain) HasProject(r *http.Request) bool {
	if d == nil {
		return false
	}

	if d.isCustomDomain() {
		return true
	}

	return d.GroupConfig.ProjectExists(r)
}

// EnsureCertificate parses the PEM-encoded certificate for the domain
func (d *Domain) EnsureCertificate() (*tls.Certificate, error) {
	if !d.isCustomDomain() {
		return nil, errors.New("tls certificates can be loaded only for pages with configuration")
	}

	d.certificateOnce.Do(func() {
		var cert tls.Certificate
		cert, d.certificateError = tls.X509KeyPair([]byte(d.Certificate), []byte(d.Key))
		if d.certificateError == nil {
			d.certificate = &cert
		}
	})

	return d.certificate, d.certificateError
}

// ServeFileHTTP returns true if something was served, false if not.
func (d *Domain) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	if d == nil {
		httperrors.Serve404(w)
		return true
	}

	if !d.IsAccessControlEnabled(r) {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	return d.Serving.ServeFileHTTP(w, r)
}

// ServeNotFoundHTTP serves the not found pages from the projects.
func (d *Domain) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	if d == nil {
		httperrors.Serve404(w)
		return
	}

	d.Serving.ServeNotFoundHTTP(w, r)
}
