package acceptance_test

import (
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrometheusMetricsCanBeScraped(t *testing.T) {
	skipUnlessEnabled(t)

	_, cleanup := newZipFileServerURL(t, "../../shared/pages/group/zip.gitlab.io/public.zip")
	defer cleanup()

	_, teardown := RunPagesProcessWithStubGitLabServer(t, true, *pagesBinary, supportedListeners(), ":42345", []string{}, "-max-conns=10")
	defer teardown()

	// need to call an actual resource to populate certain metrics e.g. gitlab_pages_domains_source_api_requests_total
	res, err := GetPageFromListener(t, httpListener, "zip.gitlab.io",
		"/symlink.html")
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, res.StatusCode)

	resp, err := http.Get("http://localhost:42345/metrics")
	require.NoError(t, err)

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Contains(t, string(body), "gitlab_pages_http_in_flight_requests 0")
	// TODO: remove metrics for disk source https://gitlab.com/gitlab-org/gitlab-pages/-/issues/382
	require.Contains(t, string(body), "gitlab_pages_served_domains 0")
	require.Contains(t, string(body), "gitlab_pages_domains_failed_total 0")
	require.Contains(t, string(body), "gitlab_pages_domains_updated_total 0")
	require.Contains(t, string(body), "gitlab_pages_last_domain_update_seconds gauge")
	require.Contains(t, string(body), "gitlab_pages_domains_configuration_update_duration gauge")
	// end TODO
	require.Contains(t, string(body), "gitlab_pages_domains_source_cache_hit")
	require.Contains(t, string(body), "gitlab_pages_domains_source_cache_miss")
	require.Contains(t, string(body), "gitlab_pages_domains_source_failures_total")
	require.Contains(t, string(body), "gitlab_pages_serverless_requests 0")
	require.Contains(t, string(body), "gitlab_pages_serverless_latency_sum 0")
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
