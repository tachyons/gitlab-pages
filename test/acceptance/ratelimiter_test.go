package acceptance_test

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

var ratelimitedListeners = map[string]struct {
	listener ListenSpec
	header   http.Header
	clientIP string
	// We perform requests to server while we're waiting for it to boot up,
	// successful request gets counted in IP rate limit
	includeWaitRequest bool
}{
	"http_listener": {
		listener:           httpListener,
		clientIP:           "127.0.0.1",
		includeWaitRequest: true,
	},
	"https_listener": {
		listener:           httpsListener,
		clientIP:           "127.0.0.1",
		includeWaitRequest: true,
	},
	"proxy_listener": {
		listener: proxyListener,
		header: http.Header{
			"X-Forwarded-For":  []string{"172.16.123.1"},
			"X-Forwarded-Host": []string{"group.gitlab-example.com"},
		},
		clientIP: "172.16.123.1",
	},
	"proxyv2_listener": {
		listener:           httpsProxyv2Listener,
		clientIP:           "10.1.1.1",
		includeWaitRequest: true,
	},
}

func TestIPRateLimits(t *testing.T) {
	testhelpers.StubFeatureFlagValue(t, feature.EnforceIPRateLimits.EnvVariable, true)

	for name, tc := range ratelimitedListeners {
		t.Run(name, func(t *testing.T) {
			rateLimit := 5
			logBuf := RunPagesProcess(t,
				withListeners([]ListenSpec{tc.listener}),
				withExtraArgument("rate-limit-source-ip", fmt.Sprint(rateLimit)),
				withExtraArgument("rate-limit-source-ip-burst", fmt.Sprint(rateLimit)),
			)

			if tc.includeWaitRequest {
				rateLimit-- // we've already used one of requests while checking if server is up
			}

			for i := 0; i < 10; i++ {
				rsp, err := GetPageFromListenerWithHeaders(t, tc.listener, "group.gitlab-example.com", "project/", tc.header)
				require.NoError(t, err)
				require.NoError(t, rsp.Body.Close())

				if i >= rateLimit {
					require.Equal(t, http.StatusTooManyRequests, rsp.StatusCode, "group.gitlab-example.com request: %d failed", i)
					assertLogFound(t, logBuf, []string{"request hit rate limit", "\"source_ip\":\"" + tc.clientIP + "\""})
				} else {
					require.Equal(t, http.StatusOK, rsp.StatusCode, "request: %d failed", i)
				}
			}
		})
	}
}

func TestDomainRateLimits(t *testing.T) {
	testhelpers.StubFeatureFlagValue(t, feature.EnforceDomainRateLimits.EnvVariable, true)

	for name, tc := range ratelimitedListeners {
		t.Run(name, func(t *testing.T) {
			rateLimit := 5
			logBuf := RunPagesProcess(t,
				withListeners([]ListenSpec{tc.listener}),
				withExtraArgument("rate-limit-domain", fmt.Sprint(rateLimit)),
				withExtraArgument("rate-limit-domain-burst", fmt.Sprint(rateLimit)),
			)

			for i := 0; i < 10; i++ {
				rsp, err := GetPageFromListenerWithHeaders(t, tc.listener, "group.gitlab-example.com", "project/", tc.header)
				require.NoError(t, err)
				require.NoError(t, rsp.Body.Close())

				if i >= rateLimit {
					require.Equal(t, http.StatusTooManyRequests, rsp.StatusCode, "group.gitlab-example.com request: %d failed", i)
					assertLogFound(t, logBuf, []string{"request hit rate limit", "\"source_ip\":\"" + tc.clientIP + "\""})
				} else {
					require.Equal(t, http.StatusOK, rsp.StatusCode, "request: %d failed", i)
				}
			}

			// make sure that requests to other domains are passing
			rsp, err := GetPageFromListener(t, tc.listener, "CapitalGroup.gitlab-example.com", "project/")
			require.NoError(t, err)
			require.NoError(t, rsp.Body.Close())

			require.Equal(t, http.StatusOK, rsp.StatusCode, "request to unrelated domain failed")
		})
	}
}

func TestTLSRateLimits(t *testing.T) {
	tests := map[string]struct {
		spec           ListenSpec
		domainLimit    bool
		sourceIP       string
		enforceEnabled bool
	}{
		"https_with_domain_limit": {
			spec:           httpsListener,
			domainLimit:    true,
			sourceIP:       "127.0.0.1",
			enforceEnabled: true,
		},
		"https_with_domain_limit_not_enforced": {
			spec:           httpsListener,
			domainLimit:    true,
			sourceIP:       "127.0.0.1",
			enforceEnabled: false,
		},
		"https_with_ip_limit": {
			spec:           httpsListener,
			sourceIP:       "127.0.0.1",
			enforceEnabled: true,
		},
		"https_with_ip_limit_not_enforced": {
			spec:           httpsListener,
			sourceIP:       "127.0.0.1",
			enforceEnabled: false,
		},
		"proxyv2_with_domain_limit": {
			spec:           httpsProxyv2Listener,
			domainLimit:    true,
			sourceIP:       "10.1.1.1",
			enforceEnabled: true,
		},
		"proxyv2_with_domain_limit_not_enforced": {
			spec:           httpsProxyv2Listener,
			domainLimit:    true,
			sourceIP:       "10.1.1.1",
			enforceEnabled: false,
		},
		"proxyv2_with_ip_limit": {
			spec:           httpsProxyv2Listener,
			sourceIP:       "10.1.1.1",
			enforceEnabled: true,
		},
		"proxyv2_with_ip_limit_not_enforced": {
			spec:           httpsProxyv2Listener,
			sourceIP:       "10.1.1.1",
			enforceEnabled: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rateLimit := 5

			options := []processOption{
				withListeners([]ListenSpec{tt.spec}),
				withExtraArgument("metrics-address", ":42345"),
			}

			featureName := feature.EnforceIPTLSRateLimits.EnvVariable
			limitName := "tls_connections_by_source_ip"

			if tt.domainLimit {
				options = append(options,
					withExtraArgument("rate-limit-tls-domain", fmt.Sprint(rateLimit)),
					withExtraArgument("rate-limit-tls-domain-burst", fmt.Sprint(rateLimit)))

				featureName = feature.EnforceDomainTLSRateLimits.EnvVariable
				limitName = "tls_connections_by_domain"
			} else {
				options = append(options,
					withExtraArgument("rate-limit-tls-source-ip", fmt.Sprint(rateLimit)),
					withExtraArgument("rate-limit-tls-source-ip-burst", fmt.Sprint(rateLimit)))
			}

			testhelpers.StubFeatureFlagValue(t, featureName, tt.enforceEnabled)
			logBuf := RunPagesProcess(t, options...)

			// when we start the process we make 1 requests to verify that process is up
			// it gets counted in the rate limit for IP, but host is different
			if !tt.domainLimit {
				rateLimit--
			}

			for i := 0; i < 10; i++ {
				rsp, err := makeTLSRequest(t, tt.spec)

				if i >= rateLimit {
					assertLogFound(t, logBuf, []string{
						"TLS connection rate-limited",
						"\"req_host\":\"group.gitlab-example.com\"",
						fmt.Sprintf("\"source_ip\":\"%s\"", tt.sourceIP),
						"\"enforced\":" + strconv.FormatBool(tt.enforceEnabled)})

					if tt.enforceEnabled {
						require.Error(t, err)
						require.Contains(t, err.Error(), "remote error: tls: internal error")
					}

					continue
				}

				require.NoError(t, err, "request: %d failed", i)
				require.NoError(t, rsp.Body.Close())
				require.Equal(t, http.StatusOK, rsp.StatusCode, "request: %d failed", i)
			}
			expectedMetric := fmt.Sprintf(
				"gitlab_pages_rate_limit_blocked_count{enforced=\"%t\",limit_name=\"%s\"} %v",
				tt.enforceEnabled, limitName, 10-rateLimit)

			RequireMetricEqual(t, "127.0.0.1:42345", expectedMetric)
		})
	}
}

func makeTLSRequest(t *testing.T, spec ListenSpec) (*http.Response, error) {
	req, err := http.NewRequest("GET", "https://group.gitlab-example.com/project", nil)
	require.NoError(t, err)

	return spec.Client().Do(req)
}

func assertLogFound(t *testing.T, logBuf *LogCaptureBuffer, expectedLogs []string) {
	t.Helper()

	// give the process enough time to write the log message
	require.Eventually(t, func() bool {
		for _, e := range expectedLogs {
			require.Contains(t, logBuf.String(), e, "log mismatch")
		}
		return true
	}, 100*time.Millisecond, 10*time.Millisecond)
}
