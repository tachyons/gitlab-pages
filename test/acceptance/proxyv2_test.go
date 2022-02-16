package acceptance_test

import (
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestProxyv2(t *testing.T) {
	logBuf := RunPagesProcess(t,
		withListeners([]ListenSpec{httpsProxyv2Listener}),
	)

	// the dummy client IP 10.1.1.1 is set by TestProxyv2Client
	tests := map[string]struct {
		host               string
		urlSuffix          string
		expectedStatusCode int
		expectedContent    string
		expectedLog        []string
	}{
		"basic_proxyv2_request": {
			host:               "group.gitlab-example.com",
			urlSuffix:          "project/",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "project-subdir\n",
			expectedLog:        []string{"\"host\":\"group.gitlab-example.com\"", "\"remote_ip\":\"10.1.1.1\""},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			logBuf.Reset()

			response, err := GetPageFromListener(t, httpsProxyv2Listener, tt.host, tt.urlSuffix)
			require.NoError(t, err)
			testhelpers.Close(t, response.Body)

			require.Equal(t, tt.expectedStatusCode, response.StatusCode)

			body, err := io.ReadAll(response.Body)
			require.NoError(t, err)

			require.Contains(t, string(body), tt.expectedContent, "content mismatch")

			// give the process enough time to write the log message
			require.Eventually(t, func() bool {
				for _, e := range tt.expectedLog {
					require.Contains(t, logBuf.String(), e, "log mismatch")
				}
				return true
			}, time.Second, time.Millisecond)
		})
	}
}
