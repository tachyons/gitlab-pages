package acceptance_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestSourceIPRateLimitMiddleware(t *testing.T) {
	testhelpers.StubFeatureFlagValue(t, feature.EnforceIPRateLimits.EnvVariable, true)

	tcs := map[string]struct {
		listener   ListenSpec
		rateLimit  float64
		rateBurst  string
		blockedIP  string
		header     http.Header
		expectFail bool
		sleep      time.Duration
	}{
		"http_slow_requests_should_not_be_blocked": {
			listener:  httpListener,
			rateLimit: 1000,
			// RunPagesProcess makes one request, so we need to allow a burst of 2
			// because r.RemoteAddr == 127.0.0.1 and X-Forwarded-For is ignored for non-proxy requests
			rateBurst: "2",
			sleep:     10 * time.Millisecond,
		},
		"https_slow_requests_should_not_be_blocked": {
			listener:  httpsListener,
			rateLimit: 1000,
			rateBurst: "2",
			sleep:     10 * time.Millisecond,
		},
		"proxy_slow_requests_should_not_be_blocked": {
			listener:  proxyListener,
			rateLimit: 1000,
			// listen-proxy uses X-Forwarded-For
			rateBurst: "1",
			header: http.Header{
				"X-Forwarded-For":  []string{"172.16.123.1"},
				"X-Forwarded-Host": []string{"group.gitlab-example.com"},
			},
			sleep: 10 * time.Millisecond,
		},
		"proxyv2_slow_requests_should_not_be_blocked": {
			listener:  httpsProxyv2Listener,
			rateLimit: 1000,
			rateBurst: "2",
			sleep:     10 * time.Millisecond,
		},
		"http_fast_requests_blocked_after_burst": {
			listener:   httpListener,
			rateLimit:  1,
			rateBurst:  "2",
			expectFail: true,
			blockedIP:  "127.0.0.1",
		},
		"https_fast_requests_blocked_after_burst": {
			listener:   httpsListener,
			rateLimit:  1,
			rateBurst:  "2",
			expectFail: true,
			blockedIP:  "127.0.0.1",
		},
		"proxy_fast_requests_blocked_after_burst": {
			listener:  proxyListener,
			rateLimit: 1,
			rateBurst: "1",
			header: http.Header{
				"X-Forwarded-For":  []string{"172.16.123.1"},
				"X-Forwarded-Host": []string{"group.gitlab-example.com"},
			},
			expectFail: true,
			blockedIP:  "172.16.123.1",
		},
		"proxyv2_fast_requests_blocked_after_burst": {
			listener:   httpsProxyv2Listener,
			rateLimit:  1,
			rateBurst:  "2",
			expectFail: true,
			// use TestProxyv2Client SourceIP
			blockedIP: "10.1.1.1",
		},
	}

	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			logBuf := RunPagesProcess(t,
				withListeners([]ListenSpec{tc.listener}),
				withExtraArgument("rate-limit-source-ip", fmt.Sprint(tc.rateLimit)),
				withExtraArgument("rate-limit-source-ip-burst", tc.rateBurst),
			)

			for i := 0; i < 5; i++ {
				rsp, err := GetPageFromListenerWithHeaders(t, tc.listener, "group.gitlab-example.com", "project/", tc.header)
				require.NoError(t, err)
				rsp.Body.Close()

				if tc.expectFail && i >= int(tc.rateLimit) {
					require.Equal(t, http.StatusTooManyRequests, rsp.StatusCode, "group.gitlab-example.com request: %d failed", i)
					assertLogFound(t, logBuf, []string{"request hit rate limit", "\"source_ip\":\"" + tc.blockedIP + "\""})
					continue
				}

				require.Equal(t, http.StatusOK, rsp.StatusCode, "request: %d failed", i)
				time.Sleep(tc.sleep)
			}
		})
	}
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
