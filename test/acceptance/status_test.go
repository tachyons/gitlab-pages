package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStatusPage(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		withExtraArgument("pages-status", "/@statuscheck"),
	)

	rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "@statuscheck")
	require.NoError(t, err)
	defer rsp.Body.Close()
	require.Equal(t, http.StatusOK, rsp.StatusCode)
}
