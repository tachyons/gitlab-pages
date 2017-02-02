package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DomainsServed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_domains_served_total",
		Help: "The total number of sites served by this Pages app",
	})

	DomainUpdates = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_updated_total",
		Help: "The total number of site updates processed since daemon start",
	})

	DomainLastUpdateTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_last_domain_update_seconds",
		Help: "UNIX timestamp of the last update",
	})

	ProcessedRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gitlab_pages_http_requests_total",
		Help: "Total number of HTTP requests done serving",
	},
		[]string{"code", "method"},
	)

	SessionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_http_sessions_active",
		Help: "The number of HTTP requests currently being processed",
	})
)

func init() {
	prometheus.MustRegister(DomainsServed)
	prometheus.MustRegister(DomainUpdates)
	prometheus.MustRegister(DomainLastUpdateTime)
	prometheus.MustRegister(ProcessedRequests)
	prometheus.MustRegister(SessionsActive)
}
