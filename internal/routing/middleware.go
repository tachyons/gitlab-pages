package routing

import (
	"context"
	"errors"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
)

// NewMiddleware returns middleware which determine the host and domain for the request, for
// downstream middlewares to use
func NewMiddleware(handler http.Handler, s source.Source) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if we could not retrieve a domain from domains source we break the
		// middleware chain and simply respond with 502 after logging this
		d, err := getDomain(r, s)
		if err != nil && !errors.Is(err, domain.ErrDomainDoesNotExist) {
			if errors.Is(err, context.Canceled) {
				httperrors.Serve404(w)
				return
			}

			logging.LogRequest(r).WithError(err).Error("could not fetch domain information from a source")

			httperrors.Serve502(w)
			return
		}

		r = domain.ReqWithDomain(r, d)

		handler.ServeHTTP(w, r)
	})
}

func getDomain(r *http.Request, s source.Source) (*domain.Domain, error) {
	host := request.GetHostWithoutPort(r)
	return s.GetDomain(r.Context(), host)
}
