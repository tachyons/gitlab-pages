package acceptance_test

import (
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestProxyv2(t *testing.T) {
	skipUnlessEnabled(t)

	logBuf, teardown := RunPagesProcessWithOutput(t, *pagesBinary, listeners, "")
	defer teardown()

	// the dummy client IP 10.1.1.1 is set by TestProxyv2Client
	tests := map[string]struct {
		host               string
		urlSuffix          string
		expectedStatusCode int
		expectedContent    string
		expectedLog        string
	}{
		"basic_proxyv2_request": {
			host:               "group.gitlab-example.com",
			urlSuffix:          "project/",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "project-subdir\n",
			expectedLog:        "group.gitlab-example.com 10.1.1.1",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			logBuf.Reset()

			response, err := GetPageFromListener(t, httpsProxyv2Listener, tt.host, tt.urlSuffix)
			require.NoError(t, err)
			defer response.Body.Close()

			require.Equal(t, tt.expectedStatusCode, response.StatusCode)

			body, err := ioutil.ReadAll(response.Body)
			require.NoError(t, err)

			require.Contains(t, string(body), tt.expectedContent, "content mismatch")

			// give the process enough time to write the log message
			require.Eventually(t, func() bool {
				require.Contains(t, logBuf.String(), tt.expectedLog, "log mismatch")
				return true
			}, time.Second, time.Millisecond)
		})
	}
}
