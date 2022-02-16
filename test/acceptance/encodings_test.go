package acceptance_test

import (
	"mime"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestMIMETypes(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	tests := map[string]struct {
		file                string
		expectedContentType string
	}{
		"manifest_json": {
			file:                "file.webmanifest",
			expectedContentType: "application/manifest+json",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rsp, err := GetPageFromListener(t, httpListener, "group.gitlab-example.com", "project/"+tt.file)
			require.NoError(t, err)
			testhelpers.Close(t, rsp.Body)

			require.Equal(t, http.StatusOK, rsp.StatusCode)
			mt, _, err := mime.ParseMediaType(rsp.Header.Get("Content-Type"))
			require.NoError(t, err)
			require.Equal(t, tt.expectedContentType, mt)
		})
	}
}

func TestCompressedEncoding(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		path     string
		encoding string
	}{
		{
			"gzip encoding",
			"group.gitlab-example.com",
			"index.html",
			"gzip",
		},
		{
			"brotli encoding",
			"group.gitlab-example.com",
			"index.html",
			"br",
		},
	}

	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := http.Header{
				"Accept-Encoding": []string{tt.encoding},
			}
			rsp, err := GetPageFromListenerWithHeaders(t, httpListener, "group.gitlab-example.com", "index.html", header)
			require.NoError(t, err)
			testhelpers.Close(t, rsp.Body)

			require.Equal(t, http.StatusOK, rsp.StatusCode)
			require.Equal(t, tt.encoding, rsp.Header.Get("Content-Encoding"))
		})
	}
}
