package local

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePath(t *testing.T) {
	ctx := context.Background()
	rootVFS, err := localVFS.Root(ctx, ".")
	require.NoError(t, err)

	root := rootVFS.(*Root)

	wd, err := os.Getwd()
	require.NoError(t, err)

	tests := map[string]struct {
		path                string
		expectedFullPath    string
		expectedVFSPath     string
		expectedInvalidPath bool
	}{
		"a valid path": {
			path:             "testdata/link",
			expectedFullPath: filepath.Join(wd, "testdata", "link"),
			expectedVFSPath:  filepath.Join("testdata", "link"),
		},
		"a path outside of root directory": {
			path:                "testdata/../../link",
			expectedInvalidPath: true,
		},
		"an absolute path": {
			// we don't support absolute paths, thus the `wd` will be preprended to `path`
			path:             filepath.Join(wd, "testdata", "link"),
			expectedFullPath: filepath.Join(wd, wd, "testdata", "link"),
			expectedVFSPath:  filepath.Join(wd, "testdata", "link")[1:], // strip leading `/`
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fullPath, vfsPath, err := root.validatePath(test.path)

			if test.expectedInvalidPath {
				require.IsType(t, &invalidPathError{}, err, "InvalidPath")
				return
			}

			require.NoError(t, err, "validatePath")
			assert.Equal(t, test.expectedFullPath, fullPath, "FullPath")
			assert.Equal(t, test.expectedVFSPath, vfsPath, "VFSPath")
		})
	}
}

func TestReadlink(t *testing.T) {
	ctx := context.Background()
	root, err := localVFS.Root(ctx, ".")
	require.NoError(t, err)

	tests := map[string]struct {
		path                string
		expectedTarget      string
		expectedErr         string
		expectedInvalidPath bool
		expectedIsNotExist  bool
	}{
		"a valid link": {
			path:           "testdata/link",
			expectedTarget: "file",
		},
		"a file": {
			path:        "testdata/file",
			expectedErr: "invalid argument",
		},
		"a path outside of root directory": {
			path:                "testdata/../../link",
			expectedInvalidPath: true,
		},
		"a non-existing link": {
			path:               "non-existing",
			expectedIsNotExist: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			target, err := root.Readlink(ctx, test.path)

			if test.expectedIsNotExist {
				require.Equal(t, test.expectedIsNotExist, errors.Is(err, fs.ErrNotExist), "IsNotExist")
				return
			}

			if test.expectedInvalidPath {
				require.IsType(t, &invalidPathError{}, err, "InvalidPath")
				return
			}

			if test.expectedErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr, "Readlink")
				return
			}

			require.NoError(t, err, "Readlink")
			assert.Equal(t, test.expectedTarget, target, "target")
		})
	}
}

func TestReadlinkAbsolutePath(t *testing.T) {
	// create structure as:
	// /tmp/dir: directory
	// /tmp/dir/symlink: points to `/tmp/file` outside of the `/tmp/dir`
	// /tmp/dir/symlink2: points to `/tmp/dir/file`
	tmpDir, cleanup := tmpDir(t)
	defer cleanup()

	dirPath := filepath.Join(tmpDir, "dir")
	err := os.Mkdir(dirPath, 0755)
	require.NoError(t, err)

	symlinkPath := filepath.Join(dirPath, "symlink")
	filePath := filepath.Join(tmpDir, "file")
	err = os.Symlink(filePath, symlinkPath)
	require.NoError(t, err)

	symlinkPath = filepath.Join(dirPath, "symlink2")
	dirFilePath := filepath.Join(dirPath, "file")
	err = os.Symlink(dirFilePath, symlinkPath)
	require.NoError(t, err)

	root, err := localVFS.Root(context.Background(), dirPath)
	require.NoError(t, err)

	tests := map[string]struct {
		path           string
		expectedTarget string
	}{
		"the absolute path is returned for file outside of `/tmp/dir": {
			path:           "symlink",
			expectedTarget: filePath,
		},
		"the relative path is returned for file inside the `/tmp/dir": {
			path:           "symlink2",
			expectedTarget: "file",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			targetPath, err := root.Readlink(context.Background(), test.path)
			require.NoError(t, err)

			assert.Equal(t, test.expectedTarget, targetPath)
		})
	}
}

func TestLstat(t *testing.T) {
	ctx := context.Background()
	root, err := localVFS.Root(ctx, ".")
	require.NoError(t, err)

	tests := map[string]struct {
		path                string
		modePerm            os.FileMode
		modeType            os.FileMode
		expectedInvalidPath bool
		expectedIsNotExist  bool
	}{
		"a directory": {
			path:     "testdata",
			modeType: os.ModeDir,
			modePerm: 0755,
		},
		"a file": {
			path:     "testdata/file",
			modeType: os.FileMode(0),
			modePerm: 0644,
		},
		"a link": {
			path:     "testdata/link",
			modeType: os.ModeSymlink,
			// modePerm: Permissions of symlinks are platform dependent
		},
		"a path outside of root directory": {
			path:                "testdata/../../link",
			expectedInvalidPath: true,
		},
		"a non-existing link": {
			path:               "non-existing",
			expectedIsNotExist: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if test.modePerm > 0 {
				require.NoError(t, os.Chmod(test.path, test.modePerm), "preparation: deterministic permissions")
			}

			fi, err := root.Lstat(ctx, test.path)

			if test.expectedIsNotExist {
				require.Equal(t, test.expectedIsNotExist, errors.Is(err, fs.ErrNotExist), "IsNotExist")
				return
			}

			if test.expectedInvalidPath {
				require.IsType(t, &invalidPathError{}, err, "InvalidPath")
				return
			}

			require.NoError(t, err, "Lstat")
			require.Equal(t, test.modeType, fi.Mode()&os.ModeType, "file mode: type")
			if test.modePerm > 0 {
				require.Equal(t, test.modePerm, fi.Mode()&os.ModePerm, "file mode: permissions")
			}
		})
	}
}

func TestOpen(t *testing.T) {
	ctx := context.Background()
	root, err := localVFS.Root(ctx, ".")
	require.NoError(t, err)

	tests := map[string]struct {
		path                string
		expectedInvalidPath bool
		expectedIsNotExist  bool
		expectedContent     string
		expectedErr         string
	}{
		"a file": {
			path:            "testdata/file",
			expectedContent: "hello\n",
		},
		"a directory": {
			path:        "testdata",
			expectedErr: errNotFile.Error(),
		},
		"a link": {
			path:        "testdata/link",
			expectedErr: "too many levels of symbolic links",
		},
		"a path outside of root directory": {
			path:                "testdata/../../link",
			expectedInvalidPath: true,
		},
		"a non-existing file": {
			path:               "non-existing",
			expectedIsNotExist: true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			file, err := root.Open(ctx, test.path)
			if file != nil {
				defer file.Close()
			}

			if test.expectedIsNotExist {
				require.Equal(t, test.expectedIsNotExist, errors.Is(err, fs.ErrNotExist), "IsNotExist")
				return
			}

			if test.expectedErr != "" {
				require.Error(t, err, "Open")
				require.Contains(t, err.Error(), test.expectedErr, "Open")
				return
			}

			if test.expectedInvalidPath {
				require.IsType(t, &invalidPathError{}, err, "InvalidPath")
				return
			}

			require.NoError(t, err, "Open")

			data, err := io.ReadAll(file)
			require.NoError(t, err, "ReadAll")
			require.Equal(t, test.expectedContent, string(data), "ReadAll")
		})
	}
}
