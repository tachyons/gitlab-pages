package disk

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_redirectPath(t *testing.T) {
	tests := map[string]struct {
		request      *http.Request
		expectedPath string
	}{
		"simple_url_no_path": {
			request:      newRequest(t, "https://domain.gitlab.io"),
			expectedPath: "//domain.gitlab.io/",
		},
		"path_only": {
			request:      newRequest(t, "https://domain.gitlab.io/index.html"),
			expectedPath: "//domain.gitlab.io/index.html/",
		},
		"query_only": {
			request:      newRequest(t, "https://domain.gitlab.io?query=test"),
			expectedPath: "//domain.gitlab.io/?query=test",
		},
		"empty_query": {
			request:      newRequest(t, "https://domain.gitlab.io?"),
			expectedPath: "//domain.gitlab.io/",
		},
		"fragment_only": {
			request:      newRequest(t, "https://domain.gitlab.io#fragment"),
			expectedPath: "//domain.gitlab.io/#fragment",
		},
		"path_and_query": {
			request:      newRequest(t, "https://domain.gitlab.io/index.html?query=test"),
			expectedPath: "//domain.gitlab.io/index.html/?query=test",
		},
		"path_and_fragment": {
			request:      newRequest(t, "https://domain.gitlab.io/index.html#fragment"),
			expectedPath: "//domain.gitlab.io/index.html/#fragment",
		},
		"query_and_fragment": {
			request:      newRequest(t, "https://domain.gitlab.io?query=test#fragment"),
			expectedPath: "//domain.gitlab.io/?query=test#fragment",
		},
		"path_query_and_fragment": {
			request:      newRequest(t, "https://domain.gitlab.io/index.html?query=test#fragment"),
			expectedPath: "//domain.gitlab.io/index.html/?query=test#fragment",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			got := redirectPath(test.request)
			require.Equal(t, test.expectedPath, got)
		})
	}
}

func newRequest(t *testing.T, url string) *http.Request {
	t.Helper()

	r, err := http.NewRequest("GET", url, nil)
	require.NoError(t, err)

	return r
}
