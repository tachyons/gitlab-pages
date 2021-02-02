package jail_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/jail"
)

func TestJailCreateAndCleanSubPaths(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("skipping test for non root user")
	}

	cage := jail.CreateTimestamped("gitlab-pages", 0755)
	subPath := "/pages/sub/path"

	// Bind mount shared folder
	cage.MkDirAll(subPath, 0755)
	cage.Bind(subPath, cage.Path()+subPath)

	require.NoError(t, cage.Build())
	require.NoError(t, cage.Dispose())

	_, err := os.Stat(cage.Path())
	require.True(t, os.IsNotExist(err), "jail was removed")
}
