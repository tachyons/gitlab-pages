package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnknownHTTPMethod(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	req, err := http.NewRequest("UNKNOWN", httpListener.URL("", ""), nil)
	require.NoError(t, err)
	req.Host = ""

	resp, err := DoPagesRequest(t, httpListener, req)
	require.NoError(t, err)
	defer resp.Body.Close()

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}
