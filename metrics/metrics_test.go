package metrics

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func TestMetricsVectorsCanBeScraped(t *testing.T) {
	reg := prometheus.NewRegistry()

	// vectors will only be available in /metrics after a label has been set/incremented so we can't test these in
	// TestPrometheusMetricsCanBeScraped as part of the acceptance tests
	reg.MustRegister(
		DomainsSourceAPIReqTotal,
		DomainsSourceAPICallDuration,
		ZipFileServingReqTotal,
		ZipFileServingReqDuration,
	)

	handler := promhttp.HandlerFor(reg, promhttp.HandlerOpts{})
	testServer := httptest.NewServer(handler)
	defer testServer.Close()

	DomainsSourceAPICallDuration.WithLabelValues("200").Set(float64(20 * time.Millisecond))
	DomainsSourceAPIReqTotal.WithLabelValues("200").Inc()

	c, err := DomainsSourceAPIReqTotal.GetMetricWithLabelValues("200")
	require.NoError(t, err)
	require.Equal(t, float64(1), testutil.ToFloat64(c))

	ZipFileServingReqDuration.WithLabelValues("200").Set(float64(20 * time.Millisecond))
	ZipFileServingReqTotal.WithLabelValues("200").Inc()

	c, err = ZipFileServingReqTotal.GetMetricWithLabelValues("200")
	require.NoError(t, err)
	require.Equal(t, float64(1), testutil.ToFloat64(c))

	metricFamilies, err := reg.Gather()
	require.NoError(t, err)

	require.Len(t, metricFamilies, 4)

	res, err := http.Get(testServer.URL + "/metrics")
	require.NoError(t, err)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)

	require.Contains(t, string(body), `gitlab_pages_domains_source_api_requests_total{status_code="200"}`)
	require.Contains(t, string(body), `gitlab_pages_domains_source_api_call_duration{status_code="200"}`)
	require.Contains(t, string(body), `gitlab_pages_httprange_zip_reader_requests_total{status_code="200"}`)
	require.Contains(t, string(body), `gitlab_pages_httprange_zip_reader_requests_duration{status_code="200"}`)
}
