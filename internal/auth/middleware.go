package auth

import (
	"errors"
	"net/http"

	domainCfg "gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/errortracking"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
)

// AuthenticationMiddleware handles authentication requests
func (a *Auth) AuthenticationMiddleware(handler http.Handler, s source.Source) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.TryAuthenticate(w, r, s) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// AuthorizationMiddleware handles authorization
func (a *Auth) AuthorizationMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := domainCfg.FromRequest(r)

		lp, err := domain.GetLookupPath(r)
		if err != nil {
			if errors.Is(err, gitlab.ErrDiskDisabled) {
				errortracking.CaptureErrWithReqAndStackTrace(err, r)
				httperrors.Serve500(w)
				return
			}

			// redirect to auth and serve not found
			a.checkAuthAndServeNotFound(domain, w, r)
			return
		}

		// This is not auth related but there's no point in having
		// an extra middleware just for this.
		if lp.IsHTTPSOnly && !request.IsHTTPS(r) {
			redirectToHTTPS(w, r, http.StatusMovedPermanently)
			return
		}

		// Only for projects that have access control enabled
		if lp.HasAccessControl {
			// accessControlMiddleware
			if a.CheckAuthentication(w, r, domain) {
				return
			}
		}

		handler.ServeHTTP(w, r)
	})
}

// checkAuthAndServeNotFound performs the auth process if domain can't be found
// the main purpose of this process is to avoid leaking the project existence/not-existence
// by behaving the same if user has no access to the project or if project simply does not exists
func (a *Auth) checkAuthAndServeNotFound(domain *domainCfg.Domain, w http.ResponseWriter, r *http.Request) {
	// To avoid user knowing if pages exist, we will force user to login and authorize pages
	if a.CheckAuthenticationWithoutProject(w, r, domain) {
		return
	}

	// auth succeeded try to serve the correct 404 page
	domain.ServeNotFoundAuthFailed(w, r)
}

func redirectToHTTPS(w http.ResponseWriter, r *http.Request, statusCode int) {
	u := *r.URL
	u.Scheme = request.SchemeHTTPS
	u.Host = r.Host
	u.User = nil

	http.Redirect(w, r, u.String(), statusCode)
}
