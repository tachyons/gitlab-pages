package ratelimiter

import (
	"net"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

// DomainRateLimiter middleware ensures that the requested domain can be served by the current
// rate limit. See -rate-limiter
func DomainRateLimiter(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := getHost(r)

			if host != "127.0.0.1" && !rl.DomainAllowed(host) {
				httperrors.Serve429(w)
				return
			}

			handler.ServeHTTP(w, r)
		})
	}
}

func getHost(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	return host
}
