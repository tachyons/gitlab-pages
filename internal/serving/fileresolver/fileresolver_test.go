package fileresolver

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenFileFromDisk(t *testing.T) {
	cleanup := setUpTests(t)
	defer cleanup()

	tests := []struct {
		name             string
		lookupPath       string
		subPath          string
		expectedFullPath string
		expectedContent  string
		expectedErr      error
	}{
		{
			name:             "file_exists_with_subpath_and_extension",
			lookupPath:       "group/group.test.io/public/",
			subPath:          "index.html",
			expectedFullPath: "group/group.test.io/public/index.html",
			expectedContent:  "main-dir\n",
		},
		{
			name:             "file_exists_without_extension",
			lookupPath:       "group/group.test.io/public/",
			subPath:          "index",
			expectedFullPath: "group/group.test.io/public/index.html",
			expectedContent:  "main-dir\n",
		},
		{
			name:             "file_exists_without_subpath",
			lookupPath:       "group/group.test.io/public/",
			subPath:          "",
			expectedFullPath: "group/group.test.io/public/index.html",
			expectedContent:  "main-dir\n",
		},
		{
			name:        "file_does_not_exist_without_subpath",
			lookupPath:  "group.no.projects/",
			subPath:     "",
			expectedErr: errFileNotFound,
		},
		{
			name:        "file_does_not_exist",
			lookupPath:  "group/group.test.io/public/",
			subPath:     "unknown_file.html",
			expectedErr: errFileNotFound,
		},
		{
			name:             "symlink_inside_public",
			lookupPath:       "group/symlink/public/",
			subPath:          "index.html",
			expectedFullPath: "group/symlink/public/content/index.html",
			expectedContent:  "group/symlink/public/content/index.html\n",
		},
		{
			name:        "symlink_outside_of_public_dir",
			lookupPath:  "group/symlink/public/",
			subPath:     "outside.html",
			expectedErr: errFileNotInPublicDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := OpenFile(tt.lookupPath, tt.subPath, filepath.EvalSymlinks, openFSFile)
			if tt.expectedErr != nil {
				require.Equal(t, tt.expectedErr, err)
				return
			}
			require.NoError(t, err)

			content, err := ioutil.ReadAll(file)
			require.NoError(t, err)
			require.Contains(t, string(content), tt.expectedContent)
		})
	}
}

func setUpTests(t *testing.T) func() {
	t.Helper()

	return chdirInPath(t, "../../../shared/pages")
}

func chdirInPath(t *testing.T, path string) func() {
	t.Helper()

	cwd, err := os.Getwd()
	require.NoError(t, err, "Cannot Getwd")

	err = os.Chdir(path)
	require.NoError(t, err, "Cannot Chdir")

	return func() {
		err := os.Chdir(cwd)
		require.NoError(t, err, "Cannot Chdir in cleanup")
	}
}

func openFSFile(fullPath string) (io.ReadCloser, error) {
	fi, err := os.Lstat(fullPath)
	if err != nil {
		return nil, errFileNotFound
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return nil, errIsDirectory
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return nil, errNotRegularFile
	}

	return os.Open(fullPath)
}
