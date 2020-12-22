package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnknownHTTPMethod(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, listeners, "")
	defer teardown()

	req, err := http.NewRequest("UNKNOWN", listeners[0].URL(""), nil)
	require.NoError(t, err)
	req.Host = ""

	resp, err := DoPagesRequest(t, req)
	require.NoError(t, err)

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}
