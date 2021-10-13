package ratelimiter

import (
	"net/http"
	"os"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

const (
	headerGitLabRealIP    = "GitLab-Real-IP"
	headerXForwardedFor   = "X-Forwarded-For"
	headerXForwardedProto = "X-Forwarded-Proto"
)

// SourceIPLimiter returns middleware for rate-limiting clients based on their IP
func (rl *RateLimiter) SourceIPLimiter(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, sourceIP := request.GetHostWithoutPort(r), request.GetRemoteAddrWithoutPort(r)
		if !rl.SourceIPAllowed(sourceIP) {
			rl.logSourceIP(r, host, sourceIP)

			// Only drop requests once FF_ENABLE_RATE_LIMITER is enabled
			// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/629
			if rateLimiterEnabled() {
				rl.sourceIPBlockedCount.WithLabelValues("true").Inc()
				httperrors.Serve429(w)
				return
			}

			rl.sourceIPBlockedCount.WithLabelValues("false").Inc()
		}

		handler.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) logSourceIP(r *http.Request, host, sourceIP string) {
	log.WithFields(logrus.Fields{
		"handler":                       "source_ip_rate_limiter",
		"correlation_id":                correlation.ExtractFromContext(r.Context()),
		"req_scheme":                    r.URL.Scheme,
		"req_host":                      r.Host,
		"req_path":                      r.URL.Path,
		"pages_domain":                  host,
		"remote_addr":                   r.RemoteAddr,
		"source_ip":                     sourceIP,
		"x_forwarded_proto":             r.Header.Get(headerXForwardedProto),
		"x_forwarded_for":               r.Header.Get(headerXForwardedFor),
		"gitlab_real_ip":                r.Header.Get(headerGitLabRealIP),
		"rate_limiter_enabled":          rateLimiterEnabled(),
		"rate_limiter_limit_per_second": rl.sourceIPLimitPerSecond,
		"rate_limiter_burst_size":       rl.sourceIPBurstSize,
	}). // TODO: change to Debug with https://gitlab.com/gitlab-org/gitlab-pages/-/issues/629
		Info("source IP hit rate limit")
}

// TODO: remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/629
func rateLimiterEnabled() bool {
	return os.Getenv("FF_ENABLE_RATE_LIMITER") == "true"
}
