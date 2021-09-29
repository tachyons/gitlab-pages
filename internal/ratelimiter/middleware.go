package ratelimiter

import (
	"net"
	"net/http"
	"os"

	"github.com/sebest/xff"
	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
)

// SourceIPLimiter middleware ensures that the originating
// rate limit. See -rate-limiter
func (rl *RateLimiter) SourceIPLimiter(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, ip, https := getReqDetails(r)

		// http requests do not contain real IP information yet
		if !rl.SourceIPAllowed(ip) && https {
			logging.LogRequest(r).WithFields(logrus.Fields{
				"handler":                       "source_ip_rate_limiter",
				"pages_domain":                  host,
				"pages_https":                   https,
				"source_ip":                     ip,
				"rate_limiter_enabled":          rateLimiterEnabled(),
				"rate_limiter_limit_per_second": rl.sourceIPLimitPerSecond,
				"rate_limiter_burst_size":       rl.sourceIPBurstSize,
			}).Info("source IP hit rate limit")

			// Only drop requests once FF_ENABLE_RATE_LIMITER is enabled
			if rateLimiterEnabled() {
				httperrors.Serve429(w)
				return
			}
		}

		handler.ServeHTTP(w, r)
	})
}

func getReqDetails(r *http.Request) (string, string, bool) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	https := r.URL.Scheme == "https"
	ip := xff.GetRemoteAddr(r)

	return host, ip, https
}

func rateLimiterEnabled() bool {
	return os.Getenv("FF_ENABLE_RATE_LIMITER") == "true"
}
