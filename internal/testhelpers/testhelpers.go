package testhelpers

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
)

var CacheConfig = config.Cache{
	CacheExpiry:          time.Second,
	CacheCleanupInterval: time.Second / 2,
	EntryRefreshTimeout:  time.Second / 2,
	RetrievalTimeout:     time.Second,
	MaxRetrievalInterval: time.Second / 3,
	MaxRetrievalRetries:  3,
}

// AssertRedirectTo asserts that handler redirects to particular URL
func AssertRedirectTo(t *testing.T, handler http.HandlerFunc, method string,
	url string, values url.Values, expectedURL string) {
	require.HTTPRedirect(t, handler, method, url, values)

	recorder := httptest.NewRecorder()

	req, _ := http.NewRequest(method, url, nil)
	req.URL.RawQuery = values.Encode()

	handler(recorder, req)

	require.Equal(t, expectedURL, recorder.Header().Get("Location"))
}

// AssertLogContains checks that wantLogEntry is contained in at least one of the log entries
func AssertLogContains(t *testing.T, wantLogEntry string, entries []*logrus.Entry) {
	t.Helper()

	if wantLogEntry != "" {
		messages := make([]string, len(entries))
		for k, entry := range entries {
			messages[k] = entry.Message
		}

		require.Contains(t, messages, wantLogEntry)
	}
}

// ToFileProtocol appends the file:// protocol to the current os.Getwd
// and formats path to be a full filepath
func ToFileProtocol(t *testing.T, path string) string {
	t.Helper()

	wd := Getwd(t)

	return fmt.Sprintf("file://%s/%s", wd, path)
}

// Getwd must return current working directory
func Getwd(t *testing.T) string {
	t.Helper()

	wd, err := os.Getwd()
	require.NoError(t, err)

	return wd
}

func PerformRequest(t *testing.T, handler http.Handler, r *http.Request) (int, string) {
	t.Helper()

	ww := httptest.NewRecorder()

	handler.ServeHTTP(ww, r)
	res := ww.Result()

	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)
	require.NoError(t, res.Body.Close())

	return res.StatusCode, string(b)
}

// Close will call the close function on a closer as part
// of the t.Cleanup function.
func Close(t *testing.T, c io.Closer) {
	t.Helper()

	t.Cleanup(func() {
		require.NoError(t, c.Close())
	})
}
