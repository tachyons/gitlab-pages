package httptransport

import (
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
			gaugeVec := prometheus.NewGaugeVec(prometheus.GaugeOpts{
				Name: t.Name(),
			}, []string{"status_code"})

			counterVec := prometheus.NewCounterVec(prometheus.CounterOpts{
				Name: t.Name(),
			}, []string{"status_code"})

			next := &mockRoundTripper{
				res: &http.Response{
					StatusCode: tt.statusCode,
				},
				err: tt.err,
			}

			mtr := &meteredRoundTripper{next, gaugeVec, counterVec}
			r := httptest.NewRequest("GET", "/", nil)

			res, err := mtr.RoundTrip(r)
			if tt.err != nil {
				counterCount := testutil.ToFloat64(counterVec.WithLabelValues("error"))
				require.Equal(t, float64(1), counterCount, "error")

				return
			}
			require.NoError(t, err)
			require.NotNil(t, res)

			statusCode := strconv.Itoa(res.StatusCode)
			gaugeValue := testutil.ToFloat64(gaugeVec.WithLabelValues(statusCode))
			require.Greater(t, gaugeValue, float64(0))

			counterCount := testutil.ToFloat64(counterVec.WithLabelValues(statusCode))
			require.Equal(t, float64(1), counterCount, statusCode)
		})
	}
}

type mockRoundTripper struct {
	res *http.Response
	err error
}

func (mrt *mockRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return mrt.res, mrt.err
}

func TestInternalTransportShouldHaveCustomConnectionPoolSettings(t *testing.T) {
	require.EqualValues(t, 100, InternalTransport.MaxIdleConns)
	require.EqualValues(t, 100, InternalTransport.MaxIdleConnsPerHost)
	require.EqualValues(t, 0, InternalTransport.MaxConnsPerHost)
	require.EqualValues(t, 90*time.Second, InternalTransport.IdleConnTimeout)
}
