package acceptance_test

import (
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestURILimits proves fix for https://gitlab.com/gitlab-org/gitlab-pages/-/issues/659
func TestURILimits(t *testing.T) {
	tests := map[string]struct {
		limit          string
		path           string
		expectedStatus int
	}{
		"with_disabled_limit": {
			limit:          "0",
			path:           "project/",
			expectedStatus: http.StatusOK,
		},
		"with_limit_set_to_request_length": {
			limit:          "19",
			path:           "/project/index.html",
			expectedStatus: http.StatusOK,
		},
		"with_uri_length_exceeding_the_limit": {
			limit:          "19",
			path:           "/project/index1.html",
			expectedStatus: http.StatusRequestURITooLong,
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			RunPagesProcess(t, withListeners([]ListenSpec{httpsListener}), withExtraArgument("max-uri-length", tt.limit))

			rsp, err := GetPageFromListener(t, httpsListener, "group.gitlab-example.com", tt.path)
			require.NoError(t, err)
			defer func() {
				require.NoError(t, rsp.Body.Close())
			}()

			require.Equal(t, tt.expectedStatus, rsp.StatusCode)

			b, err := io.ReadAll(rsp.Body)
			require.NoError(t, err)
			if tt.expectedStatus == http.StatusOK {
				require.Equal(t, "project-subdir\n", string(b))
			} else {
				require.Contains(t, string(b), "Request URI Too Long")
			}
		})
	}
}
