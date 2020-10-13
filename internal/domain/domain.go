package domain

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"strings"
	"sync"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// Domain is a domain that gitlab-pages can serve.
type Domain struct {
	servingRequest *serving.Request

	Name            string
	CertificateCert string
	CertificateKey  string

	Resolver Resolver

	certificate      *tls.Certificate
	certificateError error
	certificateOnce  sync.Once
}

// String implements Stringer.
func (d *Domain) String() string {
	return d.Name
}

func (d *Domain) isUnconfigured() bool {
	if d == nil {
		return true
	}

	return d.Resolver == nil
}

func (d *Domain) isSameProject(reqPath string) bool {
	return d.servingRequest != nil &&
		d.servingRequest.LookupPath != nil &&
		strings.Contains(reqPath, d.servingRequest.LookupPath.Path)
}

func (d *Domain) resolve(r *http.Request) *serving.Request {
	if d.isSameProject(r.URL.Path) {
		return d.servingRequest
	}

	// store serving.Request to avoid calling d.Resolver.Resolve multiple times
	d.servingRequest, _ = d.Resolver.Resolve(r)

	return d.servingRequest
}

// GetLookupPath returns a project details based on the request. It returns nil
// if project does not exist.
func (d *Domain) GetLookupPath(r *http.Request) *serving.LookupPath {
	if d.isUnconfigured() {
		return nil
	}

	return d.resolve(r).LookupPath
}

// IsHTTPSOnly figures out if the request should be handled with HTTPS
// only by looking at group and project level config.
func (d *Domain) IsHTTPSOnly(r *http.Request) bool {
	if lookupPath := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.IsHTTPSOnly
	}

	return false
}

// IsAccessControlEnabled figures out if the request is to a project that has access control enabled
func (d *Domain) IsAccessControlEnabled(r *http.Request) bool {
	if lookupPath := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.HasAccessControl
	}

	return false
}

// IsNamespaceProject figures out if the request is to a namespace project
func (d *Domain) IsNamespaceProject(r *http.Request) bool {
	if lookupPath := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.IsNamespaceProject
	}

	return false
}

// GetProjectID figures out what is the ID of the project user tries to access
func (d *Domain) GetProjectID(r *http.Request) uint64 {
	if lookupPath := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.ProjectID
	}

	return 0
}

// HasLookupPath figures out if the project exists that the user tries to access
func (d *Domain) HasLookupPath(r *http.Request) bool {
	return d.GetLookupPath(r) != nil
}

// EnsureCertificate parses the PEM-encoded certificate for the domain
func (d *Domain) EnsureCertificate() (*tls.Certificate, error) {
	if d.isUnconfigured() || len(d.CertificateKey) == 0 || len(d.CertificateCert) == 0 {
		return nil, errors.New("tls certificates can be loaded only for pages with configuration")
	}

	d.certificateOnce.Do(func() {
		var cert tls.Certificate
		cert, d.certificateError = tls.X509KeyPair(
			[]byte(d.CertificateCert),
			[]byte(d.CertificateKey),
		)
		if d.certificateError == nil {
			d.certificate = &cert
		}
	})

	return d.certificate, d.certificateError
}

// ServeFileHTTP returns true if something was served, false if not.
func (d *Domain) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	if !d.HasLookupPath(r) {
		// TODO: this seems to be wrong: as we should rather return false, and
		// fallback to `ServeNotFoundHTTP` to handle this case
		httperrors.Serve404(w)
		return true
	}

	return d.resolve(r).ServeFileHTTP(w, r)
}

// ServeNotFoundHTTP serves the not found pages from the projects.
func (d *Domain) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	if !d.HasLookupPath(r) {
		httperrors.Serve404(w)
		return
	}

	d.resolve(r).ServeNotFoundHTTP(w, r)
}

// serveNamespaceNotFound will try to find a parent namespace domain for a request
// that failed authentication so that we serve the custom namespace error page for
// public namespace domains
func (d *Domain) serveNamespaceNotFound(w http.ResponseWriter, r *http.Request) {
	// clone r and override the path and try to resolve the domain name
	clonedReq := r.Clone(context.Background())
	clonedReq.URL.Path = "/"

	namespaceDomain, err := d.Resolver.Resolve(clonedReq)
	if err != nil || namespaceDomain.LookupPath == nil {
		httperrors.Serve404(w)
		return
	}

	// for namespace domains that have no access control enabled
	if !namespaceDomain.LookupPath.HasAccessControl {
		namespaceDomain.ServeNotFoundHTTP(w, r)
		return
	}

	httperrors.Serve404(w)
}

// ServeNotFoundAuthFailed handler to be called when auth failed so the correct custom
// 404 page is served.
func (d *Domain) ServeNotFoundAuthFailed(w http.ResponseWriter, r *http.Request) {
	if !d.HasLookupPath(r) {
		httperrors.Serve404(w)
		return
	}
	if d.IsNamespaceProject(r) && !d.GetLookupPath(r).HasAccessControl {
		d.ServeNotFoundHTTP(w, r)
		return
	}

	d.serveNamespaceNotFound(w, r)
}
