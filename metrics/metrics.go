package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// TODO: remove disk source metrics https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
var (
	// DomainsServed counts the total number of sites served
	DomainsServed = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_served_domains",
		Help: "The number of sites served by this Pages app",
	})

	// DomainFailedUpdates counts the number of failed site updates
	DomainFailedUpdates = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_failed_total",
		Help: "The total number of site updates that have failed since daemon start",
	})

	// DomainUpdates counts the number of site updates successfully processed
	DomainUpdates = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_domains_updated_total",
		Help: "The total number of site updates successfully processed since daemon start",
	})

	// DomainLastUpdateTime is the UNIX timestamp of the last update
	DomainLastUpdateTime = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_last_domain_update_seconds",
		Help: "UNIX timestamp of the last update",
	})

	// DomainsConfigurationUpdateDuration is the time it takes to update domains configuration from disk
	DomainsConfigurationUpdateDuration = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "gitlab_pages_domains_configuration_update_duration",
		Help: "The time (in seconds) it takes to update domains configuration from disk",
	})

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

	// ServerlessRequests measures the amount of serverless invocations
	ServerlessRequests = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "gitlab_pages_serverless_requests",
		Help: "The number of total GitLab Serverless requests served",
	})

	// ServerlessLatency records serverless serving roundtrip duration
	ServerlessLatency = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name: "gitlab_pages_serverless_latency",
		Help: "Serverless serving roundtrip duration",
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

	// ObjectStorageBackendReqTotal is the number of requests made to Object Storage by zip file serving
	// Could be bigger than the number of pages served.
	ObjectStorageBackendReqTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "gitlab_pages_object_storage_backend_requests_total",
		Help: "The number of requests made to Object Storage by zip file serving with different status codes." +
			"Could be bigger than the number of requests served",
	}, []string{"status_code"})

	// ObjectStorageBackendReqDuration is the time it takes to get a response
	// from Object Storage in seconds for zip file servings
	ObjectStorageBackendReqDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "gitlab_pages_object_storage_backend_requests_duration",
			Help: "The time (in seconds) it takes to get a response from the " +
				"Object Storage provider for zip file serving",
		},
		[]string{"status_code"},
	)

	// ObjectStorageTraceDuration Object Storage request responsiveness in
	// seconds for different stages of an http request see httptrace.ClientTrace
	ObjectStorageTraceDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "gitlab_pages_object_storage_backend_httptrace_duration",
			Help: "Object Storage request tracing duration in seconds for " +
				"different connection stages (see Go's httptrace.ClientTrace)",
			Buckets: []float64{0.001, 0.005, 0.01, 0.02, 0.05, 0.100, 0.250,
				0.500, 1, 2, 5, 10, 20, 50},
		},
		[]string{"request_stage"},
	)

	// ZipServingOpenArchivesTotal is the number of zip archives that have been
	// opened
	ZipServingOpenArchivesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitlab_pages_zip_archives_total",
			Help: "The total number of zip archives that have been opened",
		},
		[]string{"state"},
	)

	// ZipServingArchiveCache is the number of zip archive cache hits/misses
	ZipServingArchiveCache = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitlab_pages_zip_archives_cache",
			Help: "The number of zip archives cache hits",
		},
		[]string{"cache"},
	)

	// ZipServingArchivesCurrentlyCached is the number of zip archives currently
	// in the cache
	ZipServingArchivesCurrentlyCached = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gitlab_pages_zip_archives_currently_cached",
			Help: "The number of zip archives currently in the cache",
		},
	)

	// ZipServingFilesPerZipArchiveCurrentlyCached ...
	ZipServingFilesPerZipArchiveCurrentlyCached = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gitlab_pages_files_per_zip_archive_currently_cached",
			Help: "The number of object storage zip archives currently in the cache",
		},
	)

	// ZipServingFilesPerArchiveTotalCount over time
	ZipServingFilesPerArchiveTotalCount = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "gitlab_pages_files_per_zip_archive_total",
			Help: "The number of files per zip archive total count",
		},
	)
)

// MustRegister collectors with the Prometheus client
func MustRegister() {
	prometheus.MustRegister(
		DomainsServed,
		DomainFailedUpdates,
		DomainUpdates,
		DomainLastUpdateTime,
		DomainsConfigurationUpdateDuration,
		DomainsSourceCacheHit,
		DomainsSourceCacheMiss,
		DomainsSourceAPIReqTotal,
		DomainsSourceAPICallDuration,
		DomainsSourceFailures,
		ServerlessRequests,
		ServerlessLatency,
		DiskServingFileSize,
		ServingTime,
		VFSOperations,
		ObjectStorageBackendReqTotal,
		ObjectStorageBackendReqDuration,
		ObjectStorageTraceDuration,
		ZipServingOpenArchivesTotal,
		ZipServingFilesPerArchiveTotalCount,
		ZipServingArchiveCache,
		ZipServingFilesPerZipArchiveCurrentlyCached,
		ZipServingArchivesCurrentlyCached,
	)
}
