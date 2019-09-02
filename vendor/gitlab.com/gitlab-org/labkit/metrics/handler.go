package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Metric names for the recorded metrics.
// These are the conventional names prometheus uses for these metrics.
const (
	httpInFlightRequestsMetricName         = "in_flight_requests"
	httpRequestsTotalMetricName            = "requests_total"
	httpRequestDurationSecondsMetricName   = "request_duration_seconds"
	httpRequestSizeBytesMetricName         = "request_size_bytes"
	httpResponseSizeBytesMetricName        = "response_size_bytes"
	httpTimeToWriteHeaderSecondsMetricName = "time_to_write_header_seconds"
)

// HandlerFactory creates handler middleware instances. Created by NewHandlerFactory.
type HandlerFactory func(next http.Handler, opts ...HandlerOption) http.Handler

// NewHandlerFactory will create a function for creating metric middlewares.
// The resulting function can be called multiple times to obtain multiple middleware
// instances.
// Each instance can be configured with different options that will be applied to the
// same underlying metrics.
func NewHandlerFactory(opts ...HandlerFactoryOption) HandlerFactory {
	factoryConfig := applyHandlerFactoryOptions(opts)
	var (
		httpInFlightRequests = prometheus.NewGauge(prometheus.GaugeOpts{
			Namespace: factoryConfig.namespace,
			Subsystem: factoryConfig.subsystem,
			Name:      httpInFlightRequestsMetricName,
			Help:      "A gauge of requests currently being served by the http server.",
		})

		httpRequestsTotal = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: factoryConfig.namespace,
				Subsystem: factoryConfig.subsystem,
				Name:      httpRequestsTotalMetricName,
				Help:      "A counter for requests to the http server.",
			},
			factoryConfig.labels,
		)

		httpRequestDurationSeconds = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: factoryConfig.namespace,
				Subsystem: factoryConfig.subsystem,
				Name:      httpRequestDurationSecondsMetricName,
				Help:      "A histogram of latencies for requests to the http server.",
				Buckets:   factoryConfig.requestDurationBuckets,
			},
			factoryConfig.labels,
		)

		httpRequestSizeBytes = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: factoryConfig.namespace,
				Subsystem: factoryConfig.subsystem,
				Name:      httpRequestSizeBytesMetricName,
				Help:      "A histogram of sizes of requests to the http server.",
				Buckets:   factoryConfig.byteSizeBuckets,
			},
			factoryConfig.labels,
		)

		httpResponseSizeBytes = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: factoryConfig.namespace,
				Subsystem: factoryConfig.subsystem,
				Name:      httpResponseSizeBytesMetricName,
				Help:      "A histogram of response sizes for requests to the http server.",
				Buckets:   factoryConfig.byteSizeBuckets,
			},
			factoryConfig.labels,
		)

		httpTimeToWriteHeaderSeconds = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: factoryConfig.namespace,
				Subsystem: factoryConfig.subsystem,
				Name:      httpTimeToWriteHeaderSecondsMetricName,
				Help:      "A histogram of request durations until the response headers are written.",
				Buckets:   factoryConfig.timeToWriteHeaderDurationBuckets,
			},
			factoryConfig.labels,
		)
	)

	prometheus.MustRegister(httpInFlightRequests)
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDurationSeconds)
	prometheus.MustRegister(httpRequestSizeBytes)
	prometheus.MustRegister(httpResponseSizeBytes)
	prometheus.MustRegister(httpTimeToWriteHeaderSeconds)

	return func(next http.Handler, handlerOpts ...HandlerOption) http.Handler {
		handlerConfig := applyHandlerOptions(handlerOpts)

		handler := next

		handler = promhttp.InstrumentHandlerCounter(httpRequestsTotal.MustCurryWith(handlerConfig.labelValues), handler)
		handler = promhttp.InstrumentHandlerDuration(httpRequestDurationSeconds.MustCurryWith(handlerConfig.labelValues), handler)
		handler = promhttp.InstrumentHandlerInFlight(httpInFlightRequests, handler)
		handler = promhttp.InstrumentHandlerRequestSize(httpRequestSizeBytes.MustCurryWith(handlerConfig.labelValues), handler)
		handler = promhttp.InstrumentHandlerResponseSize(httpResponseSizeBytes.MustCurryWith(handlerConfig.labelValues), handler)
		handler = promhttp.InstrumentHandlerTimeToWriteHeader(httpTimeToWriteHeaderSeconds.MustCurryWith(handlerConfig.labelValues), handler)

		return handler
	}
}
