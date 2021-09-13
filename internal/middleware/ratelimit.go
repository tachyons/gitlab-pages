package middleware

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/ratelimiter"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

// DomainRateLimiter middleware ensures that the requested domain can be served by the current
// rate limit. See -rate-limiter
func DomainRateLimiter(rl *ratelimiter.RateLimiter) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := request.GetHostWithoutPort(r)
			if host != "127.0.0.1" && !rl.DomainAllowed(host) {
				httperrors.Serve429(w)
				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}
