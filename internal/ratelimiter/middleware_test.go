package ratelimiter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

const (
	remoteAddr = "192.168.1.1"
)

var next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
})

func TestMiddlewareWithDifferentLimits(t *testing.T) {
	hook := testlog.NewGlobal()

	for tn, tc := range sharedTestCases {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				"rate_limiter",
				WithNow(mockNow),
				WithLimitPerSecond(tc.limit),
				WithBurstSize(tc.burstSize),
			)

			handler := rl.Middleware(next)

			for i := 0; i < tc.reqNum; i++ {
				r := requestFor(remoteAddr, "http://gitlab.com")
				code, body := testhelpers.PerformRequest(t, handler, r)

				if i < tc.burstSize {
					require.Equal(t, http.StatusNoContent, code, "req: %d failed", i)
				} else {
					// requests should fail after reaching tc.perDomainBurstPerSecond because mockNow
					// always returns the same time
					require.Equal(t, http.StatusTooManyRequests, code, "req: %d failed", i)
					require.Contains(t, body, "Too many requests.")
					assertSourceIPLog(t, remoteAddr, hook)
				}
			}
		})
	}
}

func TestMiddlewareDenyRequestsAfterBurst(t *testing.T) {
	hook := testlog.NewGlobal()
	blocked, cachedEntries, cacheReqs := newTestMetrics(t)

	tcs := map[string]struct {
		expectedStatus int
	}{
		"enabled_rate_limit_http_blocks": {
			expectedStatus: http.StatusTooManyRequests,
		},
	}

	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				"rate_limiter",
				WithCachedEntriesMetric(cachedEntries),
				WithCachedRequestsMetric(cacheReqs),
				WithBlockedCountMetric(blocked),
				WithNow(mockNow),
				WithLimitPerSecond(1),
				WithBurstSize(1),
			)

			// middleware is evaluated in reverse order
			handler := rl.Middleware(next)

			for i := 0; i < 5; i++ {
				r := requestFor(remoteAddr, "http://gitlab.com")
				code, _ := testhelpers.PerformRequest(t, handler, r)

				if i == 0 {
					require.Equal(t, http.StatusNoContent, code)
					continue
				}

				// burst is 1 and limit is 1 per second, all subsequent requests should fail
				require.Equal(t, tc.expectedStatus, code)
				assertSourceIPLog(t, remoteAddr, hook)
			}

			blockedCount := testutil.ToFloat64(blocked.WithLabelValues("rate_limiter"))
			require.Equal(t, float64(4), blockedCount, "blocked count")
			blocked.Reset()

			cachedCount := testutil.ToFloat64(cachedEntries.WithLabelValues("rate_limiter"))
			require.Equal(t, float64(1), cachedCount, "cached count")
			cachedEntries.Reset()

			cacheReqMiss := testutil.ToFloat64(cacheReqs.WithLabelValues("rate_limiter", "miss"))
			require.Equal(t, float64(1), cacheReqMiss, "miss count")
			cacheReqHit := testutil.ToFloat64(cacheReqs.WithLabelValues("rate_limiter", "hit"))
			require.Equal(t, float64(4), cacheReqHit, "hit count")
			cacheReqs.Reset()
		})
	}
}

func TestKeyFunc(t *testing.T) {
	tt := map[string]struct {
		keyFunc            KeyFunc
		firstRemoteAddr    string
		firstTarget        string
		secondRemoteAddr   string
		secondTarget       string
		expectedSecondCode int
	}{
		"rejected_by_ip": {
			keyFunc:            request.GetRemoteAddrWithoutPort,
			firstRemoteAddr:    "10.0.0.1",
			firstTarget:        "https://domain.gitlab.io",
			secondRemoteAddr:   "10.0.0.1",
			secondTarget:       "https://different.gitlab.io",
			expectedSecondCode: http.StatusTooManyRequests,
		},
		"rejected_by_ip_with_different_port": {
			keyFunc:            request.GetRemoteAddrWithoutPort,
			firstRemoteAddr:    "10.0.0.1:41000",
			firstTarget:        "https://domain.gitlab.io",
			secondRemoteAddr:   "10.0.0.1:41001",
			secondTarget:       "https://different.gitlab.io",
			expectedSecondCode: http.StatusTooManyRequests,
		},
		"rejected_by_domain": {
			keyFunc:            request.GetHostWithoutPort,
			firstRemoteAddr:    "10.0.0.1",
			firstTarget:        "https://domain.gitlab.io",
			secondRemoteAddr:   "10.0.0.2",
			secondTarget:       "https://domain.gitlab.io",
			expectedSecondCode: http.StatusTooManyRequests,
		},
		"rejected_by_domain_with_different_protocol": {
			keyFunc:            request.GetHostWithoutPort,
			firstRemoteAddr:    "10.0.0.1",
			firstTarget:        "https://domain.gitlab.io",
			secondRemoteAddr:   "10.0.0.2",
			secondTarget:       "http://domain.gitlab.io",
			expectedSecondCode: http.StatusTooManyRequests,
		},
		"domain_limiter_allows_same_ip": {
			keyFunc:            request.GetHostWithoutPort,
			firstRemoteAddr:    "10.0.0.1",
			firstTarget:        "https://domain.gitlab.io",
			secondRemoteAddr:   "10.0.0.1",
			secondTarget:       "https://different.gitlab.io",
			expectedSecondCode: http.StatusNoContent,
		},
		"ip_limiter_allows_same_domain": {
			keyFunc:            request.GetRemoteAddrWithoutPort,
			firstRemoteAddr:    "10.0.0.1",
			firstTarget:        "https://domain.gitlab.io",
			secondRemoteAddr:   "10.0.0.2",
			secondTarget:       "https://domain.gitlab.io",
			expectedSecondCode: http.StatusNoContent,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			handler := New(
				"rate_limiter",
				WithNow(mockNow),
				WithLimitPerSecond(1),
				WithBurstSize(1),
				WithKeyFunc(tc.keyFunc),
			).Middleware(next)

			r1 := httptest.NewRequest(http.MethodGet, tc.firstTarget, nil)
			r1.RemoteAddr = tc.firstRemoteAddr

			firstCode, _ := testhelpers.PerformRequest(t, handler, r1)
			require.Equal(t, http.StatusNoContent, firstCode)

			r2 := httptest.NewRequest(http.MethodGet, tc.secondTarget, nil)
			r2.RemoteAddr = tc.secondRemoteAddr
			secondCode, _ := testhelpers.PerformRequest(t, handler, r2)
			require.Equal(t, tc.expectedSecondCode, secondCode)
		})
	}
}

func assertSourceIPLog(t *testing.T, remoteAddr string, hook *testlog.Hook) {
	t.Helper()

	require.NotNil(t, hook.LastEntry())

	// source_ip that was rate limited
	require.Equal(t, remoteAddr, hook.LastEntry().Data["source_ip"])

	hook.Reset()
}

func newTestMetrics(t *testing.T) (*prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.CounterVec) {
	t.Helper()

	blockedGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: t.Name(),
		},
		[]string{"limit_name"},
	)

	cachedEntries := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: t.Name(),
	}, []string{"op"})

	cacheReqs := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: t.Name(),
	}, []string{"op", "cache"})

	return blockedGauge, cachedEntries, cacheReqs
}
