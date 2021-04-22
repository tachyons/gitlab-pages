package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnknownHTTPMethod(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, SupportedListeners(), "")
	defer teardown()

	req, err := http.NewRequest("UNKNOWN", httpListener.URL(""), nil)
	require.NoError(t, err)
	req.Host = ""

	resp, err := DoPagesRequest(t, httpListener, req)
	require.NoError(t, err)

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}
