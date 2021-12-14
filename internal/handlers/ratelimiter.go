package handlers

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/ratelimiter"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Ratelimiter configures the ratelimiter middleware
// TODO: make this unexported once https://gitlab.com/gitlab-org/gitlab-pages/-/issues/670 is done
func Ratelimiter(handler http.Handler, config *config.Config) http.Handler {
	if config.RateLimit.SourceIPLimitPerSecond == 0 {
		return handler
	}

	rl := ratelimiter.New(
		"source_ip",
		ratelimiter.WithCacheMaxSize(ratelimiter.DefaultSourceIPCacheSize),
		ratelimiter.WithCachedEntriesMetric(metrics.RateLimitSourceIPCachedEntries),
		ratelimiter.WithCachedRequestsMetric(metrics.RateLimitSourceIPCacheRequests),
		ratelimiter.WithBlockedCountMetric(metrics.RateLimitSourceIPBlockedCount),
		ratelimiter.WithLimitPerSecond(config.RateLimit.SourceIPLimitPerSecond),
		ratelimiter.WithBurstSize(config.RateLimit.SourceIPBurst),
	)

	return rl.Middleware(handler)
}
