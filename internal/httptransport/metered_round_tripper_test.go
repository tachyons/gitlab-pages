package httptransport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestReconfigureMeteredRoundTripper(t *testing.T) {
	histVec, counterVec := newTestMetrics(t)
	transport := NewTransport()
	transport.RegisterProtocol("file", http.NewFileTransport(http.Dir(".")))

	mrt := NewMeteredRoundTripper(transport, t.Name(), nil, histVec, counterVec, time.Millisecond)

	r := httptest.NewRequest("GET", "file:///testdata/file.html", nil)

	res, err := mrt.RoundTrip(r)
	require.NoError(t, err)
	testhelpers.Close(t, res.Body)

	require.Equal(t, http.StatusOK, res.StatusCode)
	body, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	require.Equal(t, "httptransport/testdata/file.html\n", string(body))

	// make sure counter still works
	statusCode := strconv.Itoa(res.StatusCode)
	counterCount := testutil.ToFloat64(counterVec.WithLabelValues(statusCode))
	require.Equal(t, float64(1), counterCount, statusCode)
}
