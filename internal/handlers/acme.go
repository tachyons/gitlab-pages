package handlers

import (
	"errors"
	"net/http"
	"net/url"

	"gitlab.com/gitlab-org/gitlab-pages/internal/acme"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

func AcmeMiddleware(handler http.Handler, s source.Source, gitlabURL string) http.Handler {
	if gitlabURL == "" {
		return handler
	}

	u, _ := url.Parse(gitlabURL)
	fn := serveFromDomain(s)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if acme.ServeAcmeChallenges(w, r, fn, u) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}

func serveFromDomain(s source.Source) acme.FallbackStrategy {
	return func(w http.ResponseWriter, r *http.Request) bool {
		d, err := s.GetDomain(r.Context(), request.GetHostWithoutPort(r))

		if err != nil && !errors.Is(err, domain.ErrDomainDoesNotExist) {
			logging.LogRequest(r).WithError(err).Error("could not fetch domain information from a source")

			httperrors.Serve502(w)
			return true
		}

		return d.ServeFileHTTP(w, r)
	}
}
