package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// ProcessedRequests is the number of HTTP requests served
	ProcessedRequests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gitlab_pages_http_requests_total",
		Help: "Total number of HTTP requests done serving",
	},
		[]string{"code", "method"},
	)

	// SessionsActive is the number of HTTP requests currently being processed
	SessionsActive = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_http_sessions_active",
		Help: "The number of HTTP requests currently being processed",
	})
)

func init() {
	prometheus.MustRegister(ProcessedRequests)
	prometheus.MustRegister(SessionsActive)
}
