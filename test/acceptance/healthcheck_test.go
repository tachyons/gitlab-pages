package acceptance_test

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHealthCheckListener(t *testing.T) {
	statusPath := "/-/readiness"
	statusAddress := "127.0.0.1:4100"

	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		withExtraArgument("pages-status", statusPath),
		withExtraArgument("status-address", statusAddress),
	)

	tcs := map[string]struct {
		path           string
		expectedStatus int
	}{
		"readiness_path": {path: statusPath, expectedStatus: http.StatusOK},
		"another_path":   {path: "/another-path", expectedStatus: http.StatusNotFound},
	}

	for tn, tc := range tcs {
		t.Run(tn, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://%s%s", statusAddress, tc.path), nil)
			require.NoError(t, err)

			res, err := QuickTimeoutHTTPSClient.Do(req)
			require.NoError(t, err)

			require.Equal(t, tc.expectedStatus, res.StatusCode)

			res2, err := GetPageFromListener(t, httpListener, "gitlab-example.com", tc.path)
			require.NoError(t, err)
			require.Equal(t, tc.expectedStatus, res2.StatusCode, "http listener must match too")
		})
	}
}
