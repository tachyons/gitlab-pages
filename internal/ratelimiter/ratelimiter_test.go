package ratelimiter

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

var (
	now          = "2021-09-13T15:00:00Z"
	validTime, _ = time.Parse(time.RFC3339, now)
)

func mockNow() time.Time {
	return validTime
}

var sharedTestCases = map[string]struct {
	sourceIPLimit     float64
	sourceIPBurstSize int
	reqNum            int
}{
	"one_request_per_second": {
		sourceIPLimit:     1,
		sourceIPBurstSize: 1,
		reqNum:            2,
	},
	"one_request_per_second_but_big_bucket": {
		sourceIPLimit:     1,
		sourceIPBurstSize: 10,
		reqNum:            11,
	},
	"three_req_per_second_bucket_size_one": {
		sourceIPLimit:     3,
		sourceIPBurstSize: 1, // max burst 1 means 1 at a time
		reqNum:            3,
	},
	"10_requests_per_second": {
		sourceIPLimit:     10,
		sourceIPBurstSize: 10,
		reqNum:            11,
	},
}

func TestSourceIPAllowed(t *testing.T) {
	t.Parallel()

	for tn, tc := range sharedTestCases {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				"rate_limiter",
				WithNow(mockNow),
				WithLimitPerSecond(tc.sourceIPLimit),
				WithBurstSize(tc.sourceIPBurstSize),
			)

			for i := 0; i < tc.reqNum; i++ {
				r := httptest.NewRequest(http.MethodGet, "https://domain.gitlab.io", nil)
				r.RemoteAddr = "172.16.123.1"

				got := rl.RequestAllowed(r)
				if i < tc.sourceIPBurstSize {
					require.Truef(t, got, "expected true for request no. %d", i)
				} else {
					// requests should fail after reaching tc.burstSize because mockNow
					// always returns the same time
					require.False(t, got, "expected false for request no. %d", i)
				}
			}
		})
	}
}

func TestSingleRateLimiterWithMultipleSourceIPs(t *testing.T) {
	rate := 10 * time.Millisecond

	rl := New(
		"rate_limiter",
		WithLimitPerSecond(float64(1/rate)),
		WithBurstSize(1),
	)

	wg := sync.WaitGroup{}

	testFn := func(ip string) func(t *testing.T) {
		return func(t *testing.T) {
			wg.Add(1)
			go func() {
				defer wg.Done()

				for i := 0; i < 5; i++ {
					r := httptest.NewRequest(http.MethodGet, "https://domain.gitlab.io", nil)
					r.RemoteAddr = ip
					got := rl.RequestAllowed(r)
					require.Truef(t, got, "expected true for request no. %d", i)
					time.Sleep(rate)
				}
			}()
		}
	}

	first := "172.16.123.10"
	t.Run(first, testFn(first))

	second := "172.16.123.20"
	t.Run(second, testFn(second))

	third := "172.16.123.30"
	t.Run(third, testFn(third))

	wg.Wait()
}

func newTestMetrics(t *testing.T) (*prometheus.GaugeVec, *prometheus.GaugeVec, *prometheus.CounterVec) {
	t.Helper()

	blockedGauge := prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: t.Name(),
		},
		[]string{"enforced"},
	)

	cachedEntries := prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: t.Name(),
	}, []string{"op"})

	cacheReqs := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: t.Name(),
	}, []string{"op", "cache"})

	return blockedGauge, cachedEntries, cacheReqs
}
