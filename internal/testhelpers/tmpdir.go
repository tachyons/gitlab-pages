package testhelpers

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/local"
)

var fs = vfs.Instrumented(&local.VFS{})

func TmpDir(tb testing.TB, pattern string) (vfs.Root, string) {
	tb.Helper()

	tmpDir, err := os.MkdirTemp("", pattern)
	require.NoError(tb, err)

	// On some systems `/tmp` can be a symlink
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	require.NoError(tb, err)

	root, err := fs.Root(context.Background(), tmpDir)
	require.NoError(tb, err)

	tb.Cleanup(func() {
		require.NoError(tb, os.RemoveAll(tmpDir))
	})

	return root, tmpDir
}
