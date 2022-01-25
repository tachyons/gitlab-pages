package testhelpers

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/local"
)

var fs = vfs.Instrumented(&local.VFS{})

func TmpDir(tb testing.TB) (vfs.Root, string) {
	tb.Helper()

	var err error
	tmpDir := tb.TempDir()

	// On some systems `/tmp` can be a symlink
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	require.NoError(tb, err)

	root, err := fs.Root(context.Background(), tmpDir, "")
	require.NoError(tb, err)

	return root, tmpDir
}
