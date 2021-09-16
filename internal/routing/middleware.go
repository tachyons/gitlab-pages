package routing

import (
	"errors"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// NewMiddleware returns middleware which determine the host and domain for the request, for
// downstream middlewares to use
func NewMiddleware(handler http.Handler, s source.Source) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if we could not retrieve a domain from domains source we break the
		// middleware chain and simply respond with 502 after logging this
		host, d, err := getHostAndDomain(r, s)
		if err != nil && !errors.Is(err, domain.ErrDomainDoesNotExist) {
			metrics.DomainsSourceFailures.Inc()
			logging.LogRequest(r).WithError(err).Error("could not fetch domain information from a source")

			httperrors.Serve502(w)
			return
		}

		r = request.WithHostAndDomain(r, host, d)

		handler.ServeHTTP(w, r)
	})
}

func getHostAndDomain(r *http.Request, s source.Source) (string, *domain.Domain, error) {
	host := request.GetHostWithoutPort(r)
	domain, err := s.GetDomain(r.Context(), host)

	return host, domain, err
}
