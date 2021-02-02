package httptransport

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/stretchr/testify/require"
)

func TestReconfigureMeteredRoundTripper(t *testing.T) {
	histVec, counterVec := newTestMetrics(t)
	mrt := NewMeteredRoundTripper(t.Name(), nil, histVec, counterVec, time.Millisecond)

	rt := ReconfigureMeteredRoundTripper(mrt, WithFileProtocol("file", http.NewFileTransport(http.Dir("."))))

	r := httptest.NewRequest("GET", "file:///testdata/", nil)

	res, err := rt.RoundTrip(r)
	require.NoError(t, err)
	defer res.Body.Close()

	require.Equal(t, http.StatusOK, res.StatusCode)
	body, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, "httptransport/testdata/index.html\n", string(body))

	// make sure counter still works
	statusCode := strconv.Itoa(res.StatusCode)
	counterCount := testutil.ToFloat64(counterVec.WithLabelValues(statusCode))
	require.Equal(t, float64(1), counterCount, statusCode)
}
