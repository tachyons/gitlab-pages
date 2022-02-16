package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestUnknownHTTPMethod(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	req, err := http.NewRequest("UNKNOWN", httpListener.URL(""), nil)
	require.NoError(t, err)
	req.Host = ""

	resp, err := DoPagesRequest(t, httpListener, req)
	require.NoError(t, err)
	testhelpers.Close(t, resp.Body)

	require.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)
}
