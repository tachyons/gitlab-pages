package jail_test

import (
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/jail"
)

func TestJailCreateAndCleanSubPaths(t *testing.T) {
	jailPath := tmpJailPath()
	subPath := "/pages/sub/path"

	require.NoError(t, os.MkdirAll(jailPath+subPath, 0755))
	defer os.RemoveAll(jailPath)

	cage := jail.CreateTimestamped("gitlab-pages", 0755)

	// Bind mount shared folder
	cage.MkDir(subPath, 0755)
	cage.Bind(subPath, subPath)

	require.NoError(t, cage.Build())
	require.NoError(t, cage.Dispose())

	_, err := os.Stat(path.Join(jailPath, subPath))
	require.True(t, os.IsNotExist(err), "%s in jail was not removed", subPath)
	_, err = os.Stat(jailPath)
	require.NoError(t, err, "/ in jail (corresponding to external directory) was removed")
}
