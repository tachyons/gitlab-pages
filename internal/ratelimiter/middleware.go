package ratelimiter

import (
	"net"
	"net/http"
	"os"

	"github.com/sebest/xff"
	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
)

const (
	headerGitLabRealIP    = "GitLab-Real-IP"
	headerXForwardedFor   = "X-Forwarded-For"
	headerXForwardedProto = "X-Forwarded-Proto"
)

// SourceIPLimiter middleware ensures that the originating
// rate limit. See -rate-limiter
func (rl *RateLimiter) SourceIPLimiter(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, sourceIP := rl.getReqDetails(r)
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

func (rl *RateLimiter) getReqDetails(r *http.Request) (string, string) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	// choose between r.RemoteAddr and X-Forwarded-For. Only uses XFF when proxied
	remoteAddr := xff.GetRemoteAddrIfAllowed(r, func(sip string) bool {
		// We enable github.com/gorilla/handlers.ProxyHeaders which sets r.RemoteAddr
		// with the value of X-Forwarded-For when --listen-proxy is set
		return rl.proxied
	})

	ip, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		ip = remoteAddr
	}

	return host, ip
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
		"proxied":                       rl.proxied,
		"x_forwarded_proto":             r.Header.Get(headerXForwardedProto),
		"x_forwarded_for":               r.Header.Get(headerXForwardedFor),
		"gitlab_real_ip":                r.Header.Get(headerGitLabRealIP),
		"rate_limiter_enabled":          rateLimiterEnabled(),
		"rate_limiter_limit_per_second": rl.sourceIPLimitPerSecond,
		"rate_limiter_burst_size":       rl.sourceIPBurstSize,
	}). // TODO: change to Debug with https://gitlab.com/gitlab-org/gitlab-pages/-/issues/629
		Info("source IP hit rate limit")
}

func rateLimiterEnabled() bool {
	return os.Getenv("FF_ENABLE_RATE_LIMITER") == "true"
}
