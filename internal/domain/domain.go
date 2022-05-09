package domain

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"sync"

	"gitlab.com/gitlab-org/gitlab-pages/internal/errortracking"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// ErrDomainDoesNotExist returned when a domain is not found or when a lookup path
// for a domain could not be resolved
var ErrDomainDoesNotExist = errors.New("domain does not exist")

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

// New creates a new domain with a resolver and existing certificates
func New(name, cert, key string, resolver Resolver) *Domain {
	return &Domain{
		Name:            name,
		CertificateCert: cert,
		CertificateKey:  key,
		Resolver:        resolver,
	}
}

// String implements Stringer.
func (d *Domain) String() string {
	return d.Name
}

func (d *Domain) resolve(r *http.Request) (*serving.Request, error) {
	if d == nil {
		return nil, ErrDomainDoesNotExist
	}

	return d.Resolver.Resolve(r)
}

// GetLookupPath returns a project details based on the request. It returns nil
// if project does not exist.
func (d *Domain) GetLookupPath(r *http.Request) (*serving.LookupPath, error) {
	servingReq, err := d.resolve(r)
	if err != nil {
		return nil, err
	}

	return servingReq.LookupPath, nil
}

// IsHTTPSOnly figures out if the request should be handled with HTTPS
// only by looking at group and project level config.
func (d *Domain) IsHTTPSOnly(r *http.Request) bool {
	if lookupPath, _ := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.IsHTTPSOnly
	}

	return false
}

// IsAccessControlEnabled figures out if the request is to a project that has access control enabled
func (d *Domain) IsAccessControlEnabled(r *http.Request) bool {
	if lookupPath, _ := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.HasAccessControl
	}

	return false
}

// IsNamespaceProject figures out if the request is to a namespace project
func (d *Domain) IsNamespaceProject(r *http.Request) bool {
	if lookupPath, _ := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.IsNamespaceProject
	}

	return false
}

// GetProjectID figures out what is the ID of the project user tries to access
func (d *Domain) GetProjectID(r *http.Request) uint64 {
	if lookupPath, _ := d.GetLookupPath(r); lookupPath != nil {
		return lookupPath.ProjectID
	}

	return 0
}

// EnsureCertificate parses the PEM-encoded certificate for the domain
func (d *Domain) EnsureCertificate() (*tls.Certificate, error) {
	if d == nil || len(d.CertificateKey) == 0 || len(d.CertificateCert) == 0 {
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
	request, err := d.resolve(r)
	if err != nil {
		if errors.Is(err, ErrDomainDoesNotExist) {
			// serve generic 404
			logging.LogRequest(r).WithError(ErrDomainDoesNotExist).Error("failed to serve the file")
			httperrors.Serve404(w)
			return true
		}

		errortracking.CaptureErrWithReqAndStackTrace(err, r)
		httperrors.Serve503(w)
		return true
	}

	return request.ServeFileHTTP(w, r)
}

// ServeNotFoundHTTP serves the not found pages from the projects.
func (d *Domain) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	request, err := d.resolve(r)
	if err != nil {
		if errors.Is(err, ErrDomainDoesNotExist) {
			// serve generic 404
			logging.LogRequest(r).WithError(ErrDomainDoesNotExist).Error("failed to serve the not found page")
			httperrors.Serve404(w)
			return
		}

		errortracking.CaptureErrWithReqAndStackTrace(err, r)
		httperrors.Serve503(w)
		return
	}

	request.ServeNotFoundHTTP(w, r)
}

// ServeNamespaceNotFound will try to find a parent namespace domain for a request
// that failed authentication so that we serve the custom namespace error page for
// public namespace domains
func (d *Domain) ServeNamespaceNotFound(w http.ResponseWriter, r *http.Request) {
	// clone r and override the path and try to resolve the domain name
	clonedReq := r.Clone(context.Background())
	clonedReq.URL.Path = "/"

	namespaceDomain, err := d.Resolver.Resolve(clonedReq)
	if err != nil {
		if errors.Is(err, ErrDomainDoesNotExist) {
			// serve generic 404
			logging.LogRequest(r).WithError(ErrDomainDoesNotExist).Error("failed while finding parent namespace domain for a request that failed authentication")
			httperrors.Serve404(w)
			return
		}

		errortracking.CaptureErrWithReqAndStackTrace(err, r)
		httperrors.Serve503(w)
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
	lookupPath, err := d.GetLookupPath(r)
	if err != nil {
		httperrors.Serve404(w)
		return
	}

	if d.IsNamespaceProject(r) && !lookupPath.HasAccessControl {
		d.ServeNotFoundHTTP(w, r)
		return
	}

	d.ServeNamespaceNotFound(w, r)
}
