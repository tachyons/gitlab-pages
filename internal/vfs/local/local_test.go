package local

import (
	"context"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadlink(t *testing.T) {
	ctx := context.Background()
	fs := VFS{}

	target, err := fs.Readlink(ctx, "testdata/link")
	require.NoError(t, err)
	require.Equal(t, "file", target)
}

func TestReadlinkNotSymlink(t *testing.T) {
	ctx := context.Background()
	fs := VFS{}

	for _, path := range []string{"testdata", "testdata/file"} {
		t.Run(path, func(t *testing.T) {
			_, err := os.Lstat(path)
			require.NoError(t, err, "sanity check: input must actually exist")

			_, err = fs.Readlink(ctx, path)
			require.Error(t, err, "expect readlink to fail")
		})
	}
}

func TestLstat(t *testing.T) {
	ctx := context.Background()
	fs := VFS{}

	testCases := []struct {
		path     string
		modePerm os.FileMode
		modeType os.FileMode
	}{
		{path: "testdata", modeType: os.ModeDir, modePerm: 0755},
		{path: "testdata/file", modeType: os.FileMode(0), modePerm: 0644},
		{path: "testdata/link", modeType: os.ModeSymlink}, // Permissions of symlinks are platform dependent
	}

	for _, tc := range testCases {
		t.Run(tc.path, func(t *testing.T) {
			if tc.modePerm > 0 {
				require.NoError(t, os.Chmod(tc.path, tc.modePerm), "preparation: deterministic permissions")
			}

			fi, err := fs.Lstat(ctx, tc.path)
			require.NoError(t, err, "lstat error")

			require.Equal(t, tc.modeType, fi.Mode()&os.ModeType, "file mode: type")
			if tc.modePerm > 0 {
				require.Equal(t, tc.modePerm, fi.Mode()&os.ModePerm, "file mode: permissions")
			}
		})
	}
}

func TestOpen(t *testing.T) {
	ctx := context.Background()
	fs := VFS{}

	f, err := fs.Open(ctx, "testdata/file")
	require.NoError(t, err, "open file")

	data, err := ioutil.ReadAll(f)
	require.NoError(t, err, "read from file")
	require.Equal(t, "hello\n", string(data), "file contents")

	require.NoError(t, f.Close(), "close file")
}

func TestOpenDenySymlink(t *testing.T) {
	ctx := context.Background()
	fs := VFS{}
	const symlinkPath = "testdata/link"

	fi, err := os.Stat(symlinkPath)
	require.NoError(t, err, "stat link")
	require.Equal(t, os.FileMode(0), fi.Mode()&os.ModeType, "sanity check: link target should be a regular file")

	fi, err = os.Lstat(symlinkPath)
	require.NoError(t, err, "lstat link")
	require.Equal(t, os.ModeSymlink, fi.Mode()&os.ModeType, "sanity check: testdata/link should be a symlink")

	_, err = fs.Open(ctx, symlinkPath)
	require.Error(t, err, "opening symlink should fail (security mechanism)")
}
