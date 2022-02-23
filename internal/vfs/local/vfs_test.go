package local

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var localVFS = &VFS{}

func tmpDir(t *testing.T) string {
	var err error
	tmpDir := t.TempDir()

	// On some systems `/tmp` can be a symlink
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	return tmpDir
}

func TestVFSRoot(t *testing.T) {
	// create structure as:
	// /tmp/dir: directory
	// /tmp/dir_link: symlink to `dir`
	// /tmp/dir_absolute_link: symlink to `/tmp/dir`
	// /tmp/file: file
	// /tmp/file_link: symlink to `file`
	// /tmp/file_absolute_link: symlink to `/tmp/file`
	tmpDir := tmpDir(t)

	dirPath := filepath.Join(tmpDir, "dir")
	err := os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	filePath := filepath.Join(tmpDir, "file")
	err = os.WriteFile(filePath, []byte{}, 0644)
	require.NoError(t, err)

	symlinks := map[string]string{
		"dir_link":           "dir",
		"dir_absolute_link":  dirPath,
		"file_link":          "file",
		"file_absolute_link": filePath,
	}

	for dest, src := range symlinks {
		err := os.Symlink(src, filepath.Join(tmpDir, dest))
		require.NoError(t, err)
	}

	tests := map[string]struct {
		path         string
		expectedPath string
		expectedErr  error
	}{
		"a valid directory": {
			path:         "dir",
			expectedPath: dirPath,
		},
		"a symlink to directory": {
			path:         "dir_link",
			expectedPath: dirPath,
		},
		"a symlink to absolute directory": {
			path:         "dir_absolute_link",
			expectedPath: dirPath,
		},
		"a file": {
			path:        "file",
			expectedErr: errNotDirectory,
		},
		"a symlink to file": {
			path:        "file_link",
			expectedErr: errNotDirectory,
		},
		"a symlink to absolute file": {
			path:        "file_absolute_link",
			expectedErr: errNotDirectory,
		},
		"a non-existing file": {
			path:        "not-existing",
			expectedErr: fs.ErrNotExist,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rootVFS, err := localVFS.Root(context.Background(), filepath.Join(tmpDir, test.path), "")

			if test.expectedErr != nil {
				require.ErrorIs(t, err, test.expectedErr)
				return
			}

			require.NoError(t, err)
			require.IsType(t, &Root{}, rootVFS)
			assert.Equal(t, test.expectedPath, rootVFS.(*Root).rootPath)
		})
	}
}
