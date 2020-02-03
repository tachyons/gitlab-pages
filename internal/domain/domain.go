package domain

import (
	"crypto/tls"
	"errors"
	"net/http"
	"sync"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
)

// Domain is a domain that gitlab-pages can serve.
type Domain struct {
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

func (d *Domain) resolve(r *http.Request) (*serving.LookupPath, string) {
	lookupPath, subpath, _ := d.Resolver.Resolve(r)

	// Current implementation does not return errors in any case
	if lookupPath == nil {
		return nil, ""
	}

	return lookupPath, subpath
}

// GetLookupPath returns a project details based on the request
func (d *Domain) GetLookupPath(r *http.Request) *serving.LookupPath {
	lookupPath, _ := d.resolve(r)

	return lookupPath
}

// Handler returns a serving handler for this request
func (d *Domain) Handler(w http.ResponseWriter, r *http.Request) serving.Handler {
	project, subpath := d.resolve(r)

	return serving.Handler{
		Writer:     w,
		Request:    r,
		LookupPath: project,
		SubPath:    subpath,
		Serving:    disk.New(),
	}
}

// IsHTTPSOnly figures out if the request should be handled with HTTPS
// only by looking at group and project level config.
func (d *Domain) IsHTTPSOnly(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	if lookupPath := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.IsHTTPSOnly
	}

	return false
}

// IsAccessControlEnabled figures out if the request is to a project that has access control enabled
func (d *Domain) IsAccessControlEnabled(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	if lookupPath := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.HasAccessControl
	}

	return false
}

// IsNamespaceProject figures out if the request is to a namespace project
func (d *Domain) IsNamespaceProject(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	if lookupPath := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.IsNamespaceProject
	}

	return false
}

// GetProjectID figures out what is the ID of the project user tries to access
func (d *Domain) GetProjectID(r *http.Request) uint64 {
	if d.isUnconfigured() {
		return 0
	}

	if lookupPath := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.ProjectID
	}

	return 0
}

// HasLookupPath figures out if the project exists that the user tries to access
func (d *Domain) HasLookupPath(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

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
	if d.isUnconfigured() || !d.HasLookupPath(r) {
		// TODO: this seems to be wrong: as we should rather return false, and
		// fallback to `ServeNotFoundHTTP` to handle this case
		httperrors.Serve404(w)
		return true
	}

	return d.Handler(w, r).ServeFileHTTP()
	//d.Serving().ServeFileHTTP(d.toHandler(w, r))
}

// ServeNotFoundHTTP serves the not found pages from the projects.
func (d *Domain) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	if d.isUnconfigured() || !d.HasLookupPath(r) {
		httperrors.Serve404(w)
		return
	}

	d.Handler(w, r).ServeNotFoundHTTP()
	//d.Serving().ServeNotFoundHTTP(d.toHandler(w, r))
}
