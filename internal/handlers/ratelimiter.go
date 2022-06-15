package handlers

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/ratelimiter"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Ratelimiter configures the ratelimiter middleware
// TODO: make this unexported once https://gitlab.com/gitlab-org/gitlab-pages/-/issues/670 is done
func Ratelimiter(handler http.Handler, config *config.RateLimit) http.Handler {
	sourceIPLimiter := ratelimiter.New(
		"http_requests_by_source_ip",
		ratelimiter.WithCacheMaxSize(ratelimiter.DefaultSourceIPCacheSize),
		ratelimiter.WithCachedEntriesMetric(metrics.RateLimitCachedEntries),
		ratelimiter.WithCachedRequestsMetric(metrics.RateLimitCacheRequests),
		ratelimiter.WithBlockedCountMetric(metrics.RateLimitBlockedCount),
		ratelimiter.WithLimitPerSecond(config.SourceIPLimitPerSecond),
		ratelimiter.WithBurstSize(config.SourceIPBurst),
	)

	handler = sourceIPLimiter.Middleware(handler)

	domainLimiter := ratelimiter.New(
		"http_requests_by_domain",
		ratelimiter.WithCacheMaxSize(ratelimiter.DefaultDomainCacheSize),
		ratelimiter.WithKeyFunc(request.GetHostWithoutPort),
		ratelimiter.WithCachedEntriesMetric(metrics.RateLimitCachedEntries),
		ratelimiter.WithCachedRequestsMetric(metrics.RateLimitCacheRequests),
		ratelimiter.WithBlockedCountMetric(metrics.RateLimitBlockedCount),
		ratelimiter.WithLimitPerSecond(config.DomainLimitPerSecond),
		ratelimiter.WithBurstSize(config.DomainBurst),
	)

	return domainLimiter.Middleware(handler)
}
