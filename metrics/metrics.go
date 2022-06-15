package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// DomainsSourceCacheHit is the number of GitLab API call cache hits
	DomainsSourceCacheHit = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_source_cache_hit",
		Help: "The number of GitLab domains API cache hits",
	})

	// DomainsSourceCacheMiss is the number of GitLab API call cache misses
	DomainsSourceCacheMiss = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_source_cache_miss",
		Help: "The number of GitLab domains API cache misses",
	})

	// DomainsSourceFailures is the number of GitLab API calls that failed
	DomainsSourceFailures = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_source_failures_total",
		Help: "The number of GitLab API calls that failed",
	})

	// DomainsSourceAPIReqTotal is the number of calls made to the GitLab API that returned a 4XX error
	DomainsSourceAPIReqTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_source_api_requests_total",
		Help: "The number of GitLab domains API calls with different status codes",
	}, []string{"status_code"})

	// DomainsSourceAPICallDuration is the time it takes to get a response from the GitLab API in seconds
	DomainsSourceAPICallDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "gitlab_pages_domains_source_api_call_duration",
		Help: "The time (in seconds) it takes to get a response from the GitLab domains API",
	}, []string{"status_code"})

	// DomainsSourceAPITraceDuration requests trace duration in seconds for
	// different stages of an http request (see httptrace.ClientTrace)
	DomainsSourceAPITraceDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "gitlab_pages_domains_source_api_trace_duration",
			Help: "Domain source API request tracing duration in seconds for " +
				"different connection stages (see Go's httptrace.ClientTrace)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.02, 0.05, 0.100, 0.250,
				0.500, 1, 2, 5, 10, 20, 50},
		},
		[]string{"request_stage"},
	)

	// DiskServingFileSize metric for file size serving. Includes a vfs_name (local or zip).
	DiskServingFileSize = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: "gitlab_pages_disk_serving_file_size_bytes",
		Help: "The size in bytes for each file that has been served",
		// From 1B to 100MB in *10 increments (1 10 100 1,000 10,000 100,000 1'000,000 10'000,000 100'000,000)
		Buckets: prometheus.ExponentialBuckets(1.0, 10.0, 9),
	}, []string{"vfs_name"})

	// ServingTime metric for time taken to find a file serving it or not found.
	ServingTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "gitlab_pages_serving_time_seconds",
		Help:    "The time (in seconds) taken to serve a file",
		Buckets: []float64{0.1, 0.5, 1, 2.5, 5, 10, 60, 180},
	})

	// VFSOperations metric for VFS operations (lstat, readlink, open)
	VFSOperations = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gitlab_pages_vfs_operations_total",
		Help: "The number of VFS operations",
	}, []string{"vfs_name", "operation", "success"})

	// HTTPRangeRequestsTotal is the number of requests made to a
	// httprange.Resource by opening and/or reading from it. Mostly used by the
	// internal/vfs/zip package to load archives from Object Storage.
	// Could be bigger than the number of pages served.
	HTTPRangeRequestsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gitlab_pages_httprange_requests_total",
		Help: "The number of requests made by the zip VFS to a Resource with " +
			"different status codes." +
			"Could be bigger than the number of requests served",
	}, []string{"status_code"})

	// HTTPRangeRequestDuration is the time it takes to get a response
	// from an httprange.Resource hosted in object storage for a request made by
	// the zip VFS
	HTTPRangeRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "gitlab_pages_httprange_requests_duration",
			Help: "The time (in seconds) it takes to get a response from " +
				"a httprange.Resource hosted in object storage for a request " +
				"made by the zip VFS",
		},
		[]string{"status_code"},
	)

	// HTTPRangeTraceDuration httprange requests duration in seconds for
	// different stages of an http request (see httptrace.ClientTrace)
	HTTPRangeTraceDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "gitlab_pages_httprange_trace_duration",
			Help: "httprange request tracing duration in seconds for " +
				"different connection stages (see Go's httptrace.ClientTrace)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.02, 0.05, 0.100, 0.250,
				0.500, 1, 2, 5, 10, 20, 50},
		},
		[]string{"request_stage"},
	)

	// HTTPRangeOpenRequests is the number of open requests made by httprange.Reader
	HTTPRangeOpenRequests = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_httprange_open_requests",
		Help: "The number of open requests made by httprange.Reader",
	})

	// ZipOpened is the number of zip archives that have been opened
	ZipOpened = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitlab_pages_zip_opened",
			Help: "The total number of zip archives that have been opened",
		},
		[]string{"state"},
	)

	// ZipCacheRequests is the number of cache hits/misses
	ZipCacheRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitlab_pages_zip_cache_requests",
			Help: "The number of zip archives cache hits/misses",
		},
		[]string{"op", "cache"},
	)

	// ZipCachedEntries is the number of entries in the cache
	ZipCachedEntries = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitlab_pages_zip_cached_entries",
			Help: "The number of entries in the cache",
		},
		[]string{"op"},
	)

	// ZipArchiveEntriesCached is the number of files per zip archive currently
	// in the cache
	ZipArchiveEntriesCached = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gitlab_pages_zip_archive_entries_cached",
			Help: "The number of files per zip archive currently in the cache",
		},
	)

	// ZipOpenedEntriesCount is the number of files per archive total count
	// over time
	ZipOpenedEntriesCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gitlab_pages_zip_opened_entries_count",
			Help: "The number of files per zip archive total count over time",
		},
	)

	RejectedRequestsCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gitlab_pages_unknown_method_rejected_requests",
			Help: "The number of requests with unknown HTTP method which were rejected",
		},
	)

	LimitListenerMaxConns = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gitlab_pages_limit_listener_max_conns",
			Help: "The maximum concurrent connections allowed by the limit listener.",
		},
	)

	LimitListenerConcurrentConns = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gitlab_pages_limit_listener_concurrent_conns",
			Help: "The number of concurrent connections.",
		},
	)

	LimitListenerWaitingConns = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gitlab_pages_limit_listener_waiting_conns",
			Help: "The number of backlogged connections waiting on concurrency limit.",
		},
	)

	// PanicRecoveredCount measures the number of times GitLab Pages has recovered from a panic
	PanicRecoveredCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gitlab_pages_panic_recovered_count",
			Help: "The number of panics the service has recovered from.",
		},
	)

	// RateLimitCacheRequests is the number of cache hits/misses
	RateLimitCacheRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitlab_pages_rate_limit_cache_requests",
			Help: "The number of source_ip cache hits/misses in the rate limiter",
		},
		[]string{"op", "cache"},
	)

	// RateLimitCachedEntries is the number of entries in the cache
	RateLimitCachedEntries = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitlab_pages_rate_limit_cached_entries",
			Help: "The number of entries in the cache",
		},
		[]string{"op"},
	)

	// RateLimitBlockedCount is the number of requests that have been blocked
	RateLimitBlockedCount = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "gitlab_pages_rate_limit_blocked_count",
			Help: "The number of requests/connections that have been blocked by rate limiter",
		},
		[]string{"limit_name"},
	)
)

// MustRegister collectors with the Prometheus client
func MustRegister() {
	prometheus.MustRegister(
		DomainsSourceCacheHit,
		DomainsSourceCacheMiss,
		DomainsSourceAPIReqTotal,
		DomainsSourceAPICallDuration,
		DomainsSourceAPITraceDuration,
		DomainsSourceFailures,
		DiskServingFileSize,
		ServingTime,
		VFSOperations,
		HTTPRangeRequestsTotal,
		HTTPRangeRequestDuration,
		HTTPRangeTraceDuration,
		HTTPRangeOpenRequests,
		ZipOpened,
		ZipOpenedEntriesCount,
		ZipCacheRequests,
		ZipArchiveEntriesCached,
		ZipCachedEntries,
		RejectedRequestsCount,
		LimitListenerMaxConns,
		LimitListenerConcurrentConns,
		LimitListenerWaitingConns,
		PanicRecoveredCount,
		RateLimitCacheRequests,
		RateLimitCachedEntries,
		RateLimitBlockedCount,
	)
}
