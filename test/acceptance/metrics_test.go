package acceptance_test

import (
	"crypto/tls"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestPrometheusMetricsCanBeScraped(t *testing.T) {
	runObjectStorage(t, "../../shared/pages/group/zip.gitlab.io/public.zip")

	RunPagesProcess(t,
		withExtraArgument("max-conns", "10"),
		withExtraArgument("metrics-address", ":42345"),
	)

	// need to call an actual resource to populate certain metrics e.g. gitlab_pages_domains_source_api_requests_total
	res, err := GetPageFromListener(t, httpListener, "zip.gitlab.io",
		"/symlink.html")
	require.NoError(t, err)
	testhelpers.Close(t, res.Body)
	require.Equal(t, http.StatusOK, res.StatusCode)

	resp, err := http.Get("http://127.0.0.1:42345/metrics")
	require.NoError(t, err)

	testhelpers.Close(t, resp.Body)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, string(body), "gitlab_pages_http_in_flight_requests 0")

	require.Contains(t, string(body), "gitlab_pages_domains_source_cache_hit")
	require.Contains(t, string(body), "gitlab_pages_domains_source_cache_miss")
	require.Contains(t, string(body), "gitlab_pages_domains_source_failures_total")
	require.Contains(t, string(body), "gitlab_pages_disk_serving_file_size_bytes_sum")
	require.Contains(t, string(body), "gitlab_pages_serving_time_seconds_sum")
	require.Contains(t, string(body), `gitlab_pages_domains_source_api_requests_total{status_code="200"}`)
	require.Contains(t, string(body), `gitlab_pages_domains_source_api_call_duration_bucket`)
	require.Contains(t, string(body), `gitlab_pages_domains_source_api_trace_duration`)
	// httprange
	require.Contains(t, string(body), `gitlab_pages_httprange_requests_total{status_code="206"}`)
	require.Contains(t, string(body), "gitlab_pages_httprange_requests_duration_bucket")
	require.Contains(t, string(body), "gitlab_pages_httprange_trace_duration")
	require.Contains(t, string(body), "gitlab_pages_httprange_open_requests")
	// zip archives
	require.Contains(t, string(body), "gitlab_pages_zip_opened")
	require.Contains(t, string(body), "gitlab_pages_zip_cache_requests")
	require.Contains(t, string(body), "gitlab_pages_zip_cached_entries")
	require.Contains(t, string(body), "gitlab_pages_zip_archive_entries_cached")
	require.Contains(t, string(body), "gitlab_pages_zip_opened_entries_count")
	// limit_listener
	require.Contains(t, string(body), "gitlab_pages_limit_listener_max_conns")
	require.Contains(t, string(body), "gitlab_pages_limit_listener_concurrent_conns")
	require.Contains(t, string(body), "gitlab_pages_limit_listener_waiting_conns")
}

func TestMetricsHTTPSConnection(t *testing.T) {
	keyFile, certFile := CreateHTTPSFixtureFiles(t)

	RunPagesProcess(t,
		withExtraArgument("metrics-address", ":42345"),
		withExtraArgument("metrics-certificate", certFile),
		withExtraArgument("metrics-key", keyFile),
	)

	tlsConfig := &tls.Config{InsecureSkipVerify: true}
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	res, err := client.Get("https://127.0.0.1:42345/metrics")
	require.NoError(t, err)
	testhelpers.Close(t, res.Body)
	require.Equal(t, http.StatusOK, res.StatusCode)

	res, err = client.Get("http://127.0.0.1:42345/metrics")
	require.NoError(t, err)
	testhelpers.Close(t, res.Body)
	require.Equal(t, http.StatusBadRequest, res.StatusCode)
}
