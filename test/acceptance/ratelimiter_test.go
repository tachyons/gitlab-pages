package acceptance_test

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"

	"github.com/stretchr/testify/require"
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

func TestDomainateLimits(t *testing.T) {
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
