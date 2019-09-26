package domain

import (
	"crypto/tls"
	"errors"
	"net/http"
	"sync"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// Domain is a domain that gitlab-pages can serve.
type Domain struct {
	Name            string
	Location        string
	CertificateCert string
	CertificateKey  string
	Customized      bool // TODO we should get rid of this

	Resolver Resolver

	lookupPaths map[string]*Project
	serving     serving.Serving

	certificate      *tls.Certificate
	certificateError error
	certificateOnce  sync.Once
}

// String implements Stringer.
func (d *Domain) String() string {
	return d.Name
}

func (d *Domain) isCustomDomain() bool {
	if d.isUnconfigured() {
		panic("project config and group config should not be nil at the same time")
	}

	return d.Customized
}

func (d *Domain) isUnconfigured() bool {
	if d == nil {
		return true
	}

	return d.Resolver == nil
}

func (d *Domain) resolve(r *http.Request) (*Project, string) {
	// TODO use lookupPaths first

	project, subpath, _ := d.Resolver.Resolve(r)
	// current implementation does not return errors in any case

	if project == nil {
		return nil, ""
	}

	return project, subpath
}

func (d *Domain) getProject(r *http.Request) *Project {
	project, _ := d.resolve(r)

	return project
}

// Serving returns domain serving driver
func (d *Domain) Serving() serving.Serving {
	if d.serving == nil {
		d.serving = serving.NewDiskServing(d.Name, d.Location)
	}

	return d.serving
}

func (d *Domain) toHandler(w http.ResponseWriter, r *http.Request) *handler {
	project, subpath := d.resolve(r)

	return &handler{
		writer:  w,
		request: r,
		project: project,
		subpath: subpath,
	}
}

func (d *Domain) hasProject(r *http.Request) bool {
	return d.getProject(r) != nil
}

// IsHTTPSOnly figures out if the request should be handled with HTTPS
// only by looking at group and project level config.
func (d *Domain) IsHTTPSOnly(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	if project := d.getProject(r); project != nil {
		return project.IsHTTPSOnly
	}

	return false
}

// IsAccessControlEnabled figures out if the request is to a project that has access control enabled
func (d *Domain) IsAccessControlEnabled(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	if project := d.getProject(r); project != nil {
		return project.HasAccessControl
	}

	return false
}

// HasAcmeChallenge checks domain directory contains particular acme challenge
func (d *Domain) HasAcmeChallenge(r *http.Request, token string) bool {
	if d.isUnconfigured() || !d.isCustomDomain() || !d.hasProject(r) {
		return false
	}

	return d.Serving().HasAcmeChallenge(d.toHandler(nil, r), token) // TODO
}

// IsNamespaceProject figures out if the request is to a namespace project
func (d *Domain) IsNamespaceProject(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	// If request is to a custom domain, we do not handle it as a namespace project
	// as there can't be multiple projects under the same custom domain
	if d.isCustomDomain() { // TODO do we need a separate path for this
		return false
	}

	if project := d.getProject(r); project != nil {
		return project.IsNamespaceProject
	}

	return false
}

// GetID figures out what is the ID of the project user tries to access
func (d *Domain) GetID(r *http.Request) uint64 {
	if d.isUnconfigured() {
		return 0
	}

	if project := d.getProject(r); project != nil {
		return project.ID
	}

	return 0
}

// HasProject figures out if the project exists that the user tries to access
func (d *Domain) HasProject(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	if project := d.getProject(r); project != nil {
		return true
	}

	return false
}

// EnsureCertificate parses the PEM-encoded certificate for the domain
func (d *Domain) EnsureCertificate() (*tls.Certificate, error) {
	// TODO check len certificates instead of custom domain!
	if d.isUnconfigured() || !d.isCustomDomain() {
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
	if d.isUnconfigured() || !d.hasProject(r) {
		httperrors.Serve404(w)
		return true
	}

	return d.Serving().ServeFileHTTP(d.toHandler(w, r))
}

// ServeNotFoundHTTP serves the not found pages from the projects.
func (d *Domain) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	if d.isUnconfigured() || !d.hasProject(r) {
		httperrors.Serve404(w)
		return
	}

	d.Serving().ServeNotFoundHTTP(d.toHandler(w, r))
}
