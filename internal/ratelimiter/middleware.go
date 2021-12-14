package ratelimiter

import (
	"net/http"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
)

const (
	headerGitLabRealIP    = "GitLab-Real-IP"
	headerXForwardedFor   = "X-Forwarded-For"
	headerXForwardedProto = "X-Forwarded-Proto"
)

// Middleware returns middleware for rate-limiting clients
func (rl *RateLimiter) Middleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rl.RequestAllowed(r) {
			rl.logRateLimitedRequest(r)

			if feature.EnforceIPRateLimits.Enabled() {
				if rl.blockedCount != nil {
					rl.blockedCount.WithLabelValues("true").Inc()
				}
				httperrors.Serve429(w)
				return
			}

			if rl.blockedCount != nil {
				rl.blockedCount.WithLabelValues("false").Inc()
			}
		}

		handler.ServeHTTP(w, r)
	})
}

func (rl *RateLimiter) logRateLimitedRequest(r *http.Request) {
	log.WithFields(logrus.Fields{
		"rate_limiter_name":             rl.name,
		"correlation_id":                correlation.ExtractFromContext(r.Context()),
		"req_scheme":                    r.URL.Scheme,
		"req_host":                      r.Host,
		"req_path":                      r.URL.Path,
		"pages_domain":                  request.GetHostWithoutPort(r),
		"remote_addr":                   r.RemoteAddr,
		"source_ip":                     request.GetRemoteAddrWithoutPort(r),
		"x_forwarded_proto":             r.Header.Get(headerXForwardedProto),
		"x_forwarded_for":               r.Header.Get(headerXForwardedFor),
		"gitlab_real_ip":                r.Header.Get(headerGitLabRealIP),
		"rate_limiter_enabled":          feature.EnforceIPRateLimits.Enabled(),
		"rate_limiter_limit_per_second": rl.limitPerSecond,
		"rate_limiter_burst_size":       rl.burstSize,
	}). // TODO: change to Debug with https://gitlab.com/gitlab-org/gitlab-pages/-/issues/629
		Info("request hit rate limit")
}
