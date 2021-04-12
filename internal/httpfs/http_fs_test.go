package httpfs

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httptransport"

	"github.com/stretchr/testify/require"
)

func TestFSOpen(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := map[string]struct {
		allowedPaths    []string
		chrootPath      string
		fileName        string
		expectedContent string
		expectedErrMsg  string
	}{
		"file_allowed_in_file_path": {
			allowedPaths:    []string{wd + "/testdata"},
			fileName:        wd + "/testdata/file1.txt",
			expectedContent: "file1.txt\n",
		},
		"file_allowed_in_file_path_subdir": {
			allowedPaths:    []string{wd + "/testdata"},
			fileName:        wd + "/testdata/subdir/file2.txt",
			expectedContent: "subdir/file2.txt\n",
		},
		"file_not_in_allowed_path": {
			allowedPaths:   []string{wd + "/testdata/subdir"},
			fileName:       wd + "/testdata/file1.txt",
			expectedErrMsg: os.ErrPermission.Error(),
		},
		"file_does_not_exist": {
			allowedPaths:   []string{wd + "/testdata"},
			fileName:       wd + "/testdata/unknown.txt",
			expectedErrMsg: "no such file or directory",
		},
		"relative_path_not_allowed": {
			allowedPaths:   []string{"testdata"},
			fileName:       "testdata/file1.txt",
			expectedErrMsg: os.ErrPermission.Error(),
		},
		"dot_dot_in_file_resolved": {
			allowedPaths:    []string{wd + "/testdata"},
			fileName:        wd + "/../httpfs/testdata/file1.txt",
			expectedContent: "file1.txt\n",
		},
		"dot_dot_in_file_resolved_not_allowed": {
			allowedPaths:   []string{wd + "/testdata/subdir"},
			fileName:       wd + "/../httpfs/testdata/file1.txt",
			expectedErrMsg: os.ErrPermission.Error(),
		},
		"chroot_path_not_allowed_when_not_in_real_chroot": {
			allowedPaths:   []string{wd + "/testdata"},
			fileName:       wd + "/testdata/file1.txt",
			expectedErrMsg: os.ErrPermission.Error(),
			chrootPath:     wd + "/testdata",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p, err := NewFileSystemPath(test.allowedPaths, test.chrootPath)
			require.NoError(t, err)

			got, err := p.Open(test.fileName)
			if test.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrMsg)
				return
			}

			require.NoError(t, err)

			content, err := ioutil.ReadAll(got)
			require.NoError(t, err)

			require.Equal(t, test.expectedContent, string(content))
		})
	}
}

func TestFileSystemPathCanServeHTTP(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := map[string]struct {
		path               string
		chrootPath         string
		fileName           string
		escapeURL          bool
		expectedStatusCode int
		expectedContent    string
	}{
		"file_exists_in_path": {
			path:               wd + "/testdata",
			fileName:           "file1.txt",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "file1.txt\n",
		},
		"file_exists_in_sub_dir_path": {
			path:               wd + "/testdata",
			fileName:           "subdir/file2.txt",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "subdir/file2.txt\n",
		},
		"file_not_allowed_in_path": {
			path:               wd + "/testdata/subdir",
			fileName:           "../file1.txt",
			expectedStatusCode: http.StatusForbidden,
			expectedContent:    "403 Forbidden\n",
		},
		"file_does_not_exist": {
			path:               wd + "/testdata",
			fileName:           "unknown.txt",
			expectedStatusCode: http.StatusNotFound,
			expectedContent:    "404 page not found\n",
		},
		"escaped_url_is_invalid": {
			path:               wd + "/testdata",
			fileName:           "file1.txt",
			escapeURL:          true,
			expectedStatusCode: http.StatusForbidden,
			expectedContent:    "403 Forbidden\n",
		},
		"dot_dot_in_URL": {
			path:               wd + "/testdata",
			fileName:           "../testdata/file1.txt",
			expectedStatusCode: http.StatusOK,
			expectedContent:    "file1.txt\n",
		},
		"dot_dot_in_URL_outside_of_allowed_path": {
			path:               wd + "/testdata",
			fileName:           "../file1.txt",
			expectedStatusCode: http.StatusForbidden,
			expectedContent:    "403 Forbidden\n",
		},
		"chroot_path_fails_in_unit_test_forbidden_when_not_in_real_chroot": {
			path:               wd + "/testdata",
			fileName:           "file1.txt",
			chrootPath:         wd + "/testdata",
			expectedStatusCode: http.StatusForbidden,
			expectedContent:    "403 Forbidden\n",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			transport := httptransport.NewTransport()
			fs, err := NewFileSystemPath([]string{test.path}, test.chrootPath)
			require.NoError(t, err)

			transport.RegisterProtocol("file", http.NewFileTransport(fs))

			client := &http.Client{
				Transport: transport,
				Timeout:   time.Second,
			}

			reqURL := "file://" + test.path + "/" + test.fileName
			if test.escapeURL {
				reqURL = url.PathEscape(reqURL)
			}

			req, err := http.NewRequest("GET", reqURL, nil)
			require.NoError(t, err)

			res, err := client.Do(req)
			require.NoError(t, err)
			defer res.Body.Close()

			require.Equal(t, test.expectedStatusCode, res.StatusCode)
			content, err := ioutil.ReadAll(res.Body)
			require.NoError(t, err)

			require.Equal(t, test.expectedContent, string(content))
		})
	}
}
