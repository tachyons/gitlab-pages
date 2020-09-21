package testhelpers

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs"
	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/local"
)

var fs = vfs.Instrumented(local.New("local"))

func TmpDir(t *testing.T, pattern string) (vfs.Root, string, func()) {
	tmpDir, err := ioutil.TempDir("", pattern)
	if t != nil {
		require.NoError(t, err)
	}

	// On some systems `/tmp` can be a symlink
	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	if t != nil {
		require.NoError(t, err)
	}

	root, err := fs.Root(context.Background(), tmpDir)
	if t != nil {
		require.NoError(t, err)
	}

	return root, tmpDir, func() {
		os.RemoveAll(tmpDir)
	}
}
