package httptransport

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"
)

func Test_withRoundTripper(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		err        error
	}{
		{
			name:       "successful_response",
			statusCode: http.StatusNoContent,
		},
		{
			name:       "error_response",
			statusCode: http.StatusForbidden,
		},
		{
			name:       "internal_error_response",
			statusCode: http.StatusInternalServerError,
		},
		{
			name:       "unhandled_status_response",
			statusCode: http.StatusPermanentRedirect,
		},
		{
			name: "client_error",
			err:  fmt.Errorf("something went wrong"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			histVec, counterVec := newTestMetrics(t)

			next := &mockRoundTripper{
				res: &http.Response{
					StatusCode: tt.statusCode,
				},
				err:     tt.err,
				timeout: time.Nanosecond,
			}

			mtr := &meteredRoundTripper{next: next, durations: histVec, counter: counterVec, ttfbTimeout: DefaultTTFBTimeout}
			r := httptest.NewRequest("GET", "/", nil)

			// nolint:bodyclose // res.Body is nil
			res, err := mtr.RoundTrip(r)
			if tt.err != nil {
				counterCount := testutil.ToFloat64(counterVec.WithLabelValues("error"))
				require.Equal(t, float64(1), counterCount, "error")

				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)

			statusCode := strconv.Itoa(res.StatusCode)
			counterCount := testutil.ToFloat64(counterVec.WithLabelValues(statusCode))
			require.Equal(t, float64(1), counterCount, statusCode)
		})
	}
}

func TestRoundTripTTFBTimeout(t *testing.T) {
	histVec, counterVec := newTestMetrics(t)

	next := &mockRoundTripper{
		res: &http.Response{
			StatusCode: http.StatusOK,
		},
		timeout: time.Millisecond,
		err:     nil,
	}

	mtr := &meteredRoundTripper{next: next, durations: histVec, counter: counterVec, ttfbTimeout: time.Nanosecond}
	req, err := http.NewRequest("GET", "https://gitlab.com", nil)
	require.NoError(t, err)

	// nolint:bodyclose // res is nil
	res, err := mtr.RoundTrip(req)
	require.Nil(t, res)
	require.True(t, errors.Is(err, context.Canceled), "context must have been canceled after ttfb timeout")
}

func newTestMetrics(t *testing.T) (*prometheus.HistogramVec, *prometheus.CounterVec) {
	t.Helper()

	histVec := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name: t.Name(),
	}, []string{"status_code"})

	counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: t.Name(),
	}, []string{"status_code"})

	return histVec, counterVec
}

type mockRoundTripper struct {
	res     *http.Response
	err     error
	timeout time.Duration
}

func (mrt *mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	select {
	case <-r.Context().Done():
		return nil, r.Context().Err()
	case <-time.After(mrt.timeout):
		return mrt.res, mrt.err
	}
}

func TestInternalTransportShouldHaveCustomConnectionPoolSettings(t *testing.T) {
	require.EqualValues(t, 100, DefaultTransport.MaxIdleConns)
	require.EqualValues(t, 100, DefaultTransport.MaxIdleConnsPerHost)
	require.EqualValues(t, 0, DefaultTransport.MaxConnsPerHost)
	require.EqualValues(t, 90*time.Second, DefaultTransport.IdleConnTimeout)
	require.EqualValues(t, 10*time.Second, DefaultTransport.TLSHandshakeTimeout)
	require.EqualValues(t, 15*time.Second, DefaultTransport.ResponseHeaderTimeout)
	require.EqualValues(t, 15*time.Second, DefaultTransport.ExpectContinueTimeout)
}
