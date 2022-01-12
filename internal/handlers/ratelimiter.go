package handlers

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
	"gitlab.com/gitlab-org/gitlab-pages/internal/ratelimiter"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Ratelimiter configures the ratelimiter middleware
// TODO: make this unexported once https://gitlab.com/gitlab-org/gitlab-pages/-/issues/670 is done
func Ratelimiter(handler http.Handler, config *config.RateLimit) http.Handler {
	sourceIPLimiter := ratelimiter.New(
		"source_ip",
		ratelimiter.WithCacheMaxSize(ratelimiter.DefaultSourceIPCacheSize),
		ratelimiter.WithCachedEntriesMetric(metrics.RateLimitSourceIPCachedEntries),
		ratelimiter.WithCachedRequestsMetric(metrics.RateLimitSourceIPCacheRequests),
		ratelimiter.WithBlockedCountMetric(metrics.RateLimitSourceIPBlockedCount),
		ratelimiter.WithLimitPerSecond(config.SourceIPLimitPerSecond),
		ratelimiter.WithBurstSize(config.SourceIPBurst),
		ratelimiter.WithEnforce(feature.EnforceIPRateLimits.Enabled()),
	)

	handler = sourceIPLimiter.Middleware(handler)

	domainLimiter := ratelimiter.New(
		"domain",
		ratelimiter.WithCacheMaxSize(ratelimiter.DefaultDomainCacheSize),
		ratelimiter.WithKeyFunc(request.GetHostWithoutPort),
		ratelimiter.WithCachedEntriesMetric(metrics.RateLimitDomainCachedEntries),
		ratelimiter.WithCachedRequestsMetric(metrics.RateLimitDomainCacheRequests),
		ratelimiter.WithBlockedCountMetric(metrics.RateLimitDomainBlockedCount),
		ratelimiter.WithLimitPerSecond(config.DomainLimitPerSecond),
		ratelimiter.WithBurstSize(config.DomainBurst),
		ratelimiter.WithEnforce(feature.EnforceDomainRateLimits.Enabled()),
	)

	return domainLimiter.Middleware(handler)
}
