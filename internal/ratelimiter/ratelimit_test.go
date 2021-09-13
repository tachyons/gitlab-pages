package ratelimiter

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func mockNow(tb testing.TB, now string) func() time.Time {
	tb.Helper()

	return func() time.Time {
		parsedT, err := time.Parse(time.RFC3339, now)
		require.NoError(tb, err)

		return parsedT
	}
}

func TestDomainAllowed(t *testing.T) {
	now := "2021-09-13T15:00:00Z"

	tcs := map[string]struct {
		now                     string
		domainRatePerSecond     float64
		perDomainBurstPerSecond int
		domain                  string
		reqNum                  int
	}{
		"some test": {
			domainRatePerSecond:     1, // 1 per second
			perDomainBurstPerSecond: 1,
			reqNum:                  1,
			domain:                  "rate.gitlab.io",
		},
	}

	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			rl := New(
				WithNow(mockNow(t, now)),
				WithDomainRatePerSecond(tc.domainRatePerSecond),
				WithDomainBurstPerSecond(tc.perDomainBurstPerSecond),
			)

			for i := 0; i < tc.reqNum; i++ {
				got := rl.DomainAllowed(tc.domain)
				require.True(t, got, "req num: %d failed", i+1)
			}

		})
	}
}
