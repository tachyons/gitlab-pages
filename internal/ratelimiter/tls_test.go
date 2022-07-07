package ratelimiter

import (
	"crypto/tls"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/sirupsen/logrus"
	testlog "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
)

func TestTLSHostnameKey(t *testing.T) {
	info := &tls.ClientHelloInfo{ServerName: "group.gitlab.io"}

	require.Equal(t, "group.gitlab.io", TLSHostnameKey(info))
}

func TestTLSClientIPKey(t *testing.T) {
	tests := []struct {
		addr     string
		expected string
	}{
		{
			"10.1.2.3:1234",
			"10.1.2.3",
		},
		{
			"[2001:db8:3333:4444:5555:6666:7777:8888]:1234",
			"2001:db8:3333:4444:5555:6666:7777:8888",
		},
	}

	for _, tt := range tests {
		addr, err := net.ResolveTCPAddr("tcp", tt.addr)
		require.NoError(t, err)
		conn := stubConn{remoteAddr: addr}
		info := &tls.ClientHelloInfo{Conn: conn}

		require.Equal(t, tt.expected, TLSClientIPKey(info))
	}
}

func TestGetCertificateMiddleware(t *testing.T) {
	tests := map[string]struct {
		useHostnameAsKey bool
		limitPerSecond   float64
		burst            int
		successfulReqCnt int
	}{
		"ip_limiter": {
			useHostnameAsKey: false,
			limitPerSecond:   0.1,
			burst:            5,
			successfulReqCnt: 5,
		},
		"hostname_limiter": {
			useHostnameAsKey: true,
			limitPerSecond:   0.1,
			burst:            5,
			successfulReqCnt: 5,
		},
		"disabled": {
			useHostnameAsKey: false,
			limitPerSecond:   0,
			burst:            5,
			successfulReqCnt: 10,
		},
		"slowly_approach_limit": {
			useHostnameAsKey: false,
			limitPerSecond:   0.2,
			burst:            5,
			successfulReqCnt: 6, // 5 * 0.2 gives another 1 request
		},
	}

	expectedCert := &tls.Certificate{}
	expectedErr := errors.New("expected error")

	getCertificate := func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
		return expectedCert, expectedErr
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			hook := testlog.NewGlobal()
			blocked, cachedEntries, cacheReqs := newTestMetrics(t)

			keyFunc := TLSClientIPKey
			if tt.useHostnameAsKey {
				keyFunc = TLSHostnameKey
			}

			rl := New("limit_name",
				WithCachedEntriesMetric(cachedEntries),
				WithCachedRequestsMetric(cacheReqs),
				WithBlockedCountMetric(blocked),
				WithNow(stubNow()),
				WithLimitPerSecond(tt.limitPerSecond),
				WithBurstSize(tt.burst),
				WithTLSKeyFunc(keyFunc))

			middlewareGetCert := rl.GetCertificateMiddleware(getCertificate)

			addr, err := net.ResolveTCPAddr("tcp", "10.1.2.3:12345")
			require.NoError(t, err)
			conn := stubConn{remoteAddr: addr}
			info := &tls.ClientHelloInfo{Conn: conn, ServerName: "group.gitlab.io"}

			for i := 0; i < tt.successfulReqCnt; i++ {
				cert, err := middlewareGetCert(info)
				require.Equal(t, expectedCert, cert)
				require.Equal(t, expectedErr, err)
			}

			// When rate-limiter disabled altogether
			if tt.limitPerSecond <= 0 {
				return
			}

			cert, err := middlewareGetCert(info)
			require.Nil(t, cert)
			require.Equal(t, err, ErrTLSRateLimited)

			require.NotNil(t, hook.LastEntry())
			require.Equal(t, "TLS connection rate-limited", hook.LastEntry().Message)
			expectedFields := logrus.Fields{
				"rate_limiter_name":             "limit_name",
				"source_ip":                     "10.1.2.3",
				"req_host":                      "group.gitlab.io",
				"rate_limiter_limit_per_second": tt.limitPerSecond,
				"rate_limiter_burst_size":       tt.burst,
			}
			require.Equal(t, expectedFields, hook.LastEntry().Data)

			// make another request with different key and expect success
			if tt.useHostnameAsKey {
				info.ServerName = "another-group.gitlab.io"
			} else {
				addr, err := net.ResolveTCPAddr("tcp", "10.10.20.30:12345")
				require.NoError(t, err)
				conn = stubConn{remoteAddr: addr}
				info.Conn = conn
			}

			cert, err = middlewareGetCert(info)
			require.Equal(t, expectedCert, cert)
			require.Equal(t, expectedErr, err)

			blockedCount := testutil.ToFloat64(blocked.WithLabelValues("limit_name"))
			require.Equal(t, float64(1), blockedCount, "blocked count")

			cachedCount := testutil.ToFloat64(cachedEntries.WithLabelValues("limit_name"))
			require.Equal(t, float64(2), cachedCount, "cached count") // 1 for first key + 1 for different one

			cacheReqMiss := testutil.ToFloat64(cacheReqs.WithLabelValues("limit_name", "miss"))
			require.Equal(t, float64(2), cacheReqMiss, "miss count") // 1 for first key + 1 for different one
			cacheReqHit := testutil.ToFloat64(cacheReqs.WithLabelValues("limit_name", "hit"))
			require.Equal(t, float64(tt.successfulReqCnt), cacheReqHit, "hit count")
		})
	}
}

func stubNow() func() time.Time {
	now := time.Now()
	return func() time.Time {
		now = now.Add(time.Second)

		return now
	}
}
