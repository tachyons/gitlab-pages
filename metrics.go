package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	domainsServed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_domains_served_total",
		Help: "The total number of sites served by this Pages app",
	})

	domainUpdates = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_updated_total",
		Help: "The total number of site updates processed since daemon start",
	})

	domainLastUpdateTime = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_last_domain_update_seconds",
		Help: "Seconds since Unix Epoc to the last update for all domains served",
	})

	processedRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gitlab_pages_http_requests_total",
		Help: "Total number of HTTP requests done serving",
	},
		[]string{"code", "method"},
	)

	sessionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_http_sessions_active",
		Help: "The number of HTTP requests currently being processed",
	})
)

func init() {
	prometheus.MustRegister(domainsServed)
	prometheus.MustRegister(domainUpdates)
	prometheus.MustRegister(domainLastUpdateTime)
	prometheus.MustRegister(processedRequests)
	prometheus.MustRegister(sessionsActive)
}
