package acceptance_test

import (
	"mime"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMIMETypes(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcessWithoutWait(t, *pagesBinary, SupportedListeners(), "")
	defer teardown()

	require.NoError(t, httpListener.WaitUntilRequestSucceeds(nil))

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
			defer rsp.Body.Close()

			require.Equal(t, http.StatusOK, rsp.StatusCode)
			mt, _, err := mime.ParseMediaType(rsp.Header.Get("Content-Type"))
			require.NoError(t, err)
			require.Equal(t, tt.expectedContentType, mt)
		})
	}
}

func TestCompressedEncoding(t *testing.T) {
	skipUnlessEnabled(t)

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

	teardown := RunPagesProcess(t, *pagesBinary, SupportedListeners(), "")
	defer teardown()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsp, err := GetCompressedPageFromListener(t, httpListener, "group.gitlab-example.com", "index.html", tt.encoding)
			require.NoError(t, err)
			defer rsp.Body.Close()

			require.Equal(t, http.StatusOK, rsp.StatusCode)
			require.Equal(t, tt.encoding, rsp.Header.Get("Content-Encoding"))
		})
	}
}
