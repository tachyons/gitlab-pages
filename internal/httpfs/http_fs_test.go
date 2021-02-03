package httpfs

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFSOpen(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := map[string]struct {
		allowedPaths    []string
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
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			p := NewFileSystemPath(test.allowedPaths)

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
