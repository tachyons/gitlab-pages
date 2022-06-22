package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestStatusPage(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@healthcheck")
	require.NoError(t, err)
	testhelpers.Close(t, rsp.Body)
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}
