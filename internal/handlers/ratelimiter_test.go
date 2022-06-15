package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

var next = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
})

func TestRatelimiter(t *testing.T) {
	tt := map[string]struct {
		firstRemoteAddr    string
		firstTarget        string
		secondRemoteAddr   string
		secondTarget       string
		expectedSecondCode int
	}{
		"rejected_by_ip": {
			firstRemoteAddr:    "10.0.0.1",
			firstTarget:        "https://domain.gitlab.io",
			secondRemoteAddr:   "10.0.0.1",
			secondTarget:       "https://different.gitlab.io",
			expectedSecondCode: http.StatusTooManyRequests,
		},
		"rejected_by_domain": {
			firstRemoteAddr:    "10.0.0.1",
			firstTarget:        "https://domain.gitlab.io",
			secondRemoteAddr:   "10.0.0.2",
			secondTarget:       "https://domain.gitlab.io",
			expectedSecondCode: http.StatusTooManyRequests,
		},
		"different_ip_and_domain_passes": {
			firstRemoteAddr:    "10.0.0.1",
			firstTarget:        "https://domain.gitlab.io",
			secondRemoteAddr:   "10.0.0.2",
			secondTarget:       "https://different.gitlab.io",
			expectedSecondCode: http.StatusNoContent,
		},
	}

	for name, tc := range tt {
		t.Run(name, func(t *testing.T) {
			conf := config.RateLimit{
				SourceIPLimitPerSecond: 0.1,
				SourceIPBurst:          1,
				DomainLimitPerSecond:   0.1,
				DomainBurst:            1,
			}

			handler := Ratelimiter(next, &conf)

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
