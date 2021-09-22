package ratelimiter

import (
	"net"
	"net/http"
	"os"

	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
)

// DomainRateLimiter middleware ensures that the requested domain can be served by the current
// rate limit. See -rate-limiter
func DomainRateLimiter(rl *RateLimiter) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host := getHost(r)

			if !rl.DomainAllowed(host) {
				logging.LogRequest(r).WithFields(logrus.Fields{
					"handler":                 "domain_rate_limiter",
					"pages_domain":            host,
					"rate_limiter_enabled":    rateLimiterEnabled(),
					"rate_limiter_frequency":  rl.perDomainFrequency,
					"rate_limiter_burst_size": rl.perDomainBurstSize,
				}).Info("domain hit rate limit")

				// Only drop requests once FF_ENABLE_RATE_LIMITER is enabled
				if rateLimiterEnabled() {
					httperrors.Serve429(w)
					return
				}
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

func rateLimiterEnabled() bool {
	return os.Getenv("FF_ENABLE_RATE_LIMITER") == "true"
}
