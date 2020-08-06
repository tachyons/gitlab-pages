package testhelpers

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func ChdirInPath(t testing.TB, path string, chdirSet *bool) func() {
	t.Helper()

	if *chdirSet {
		return func() {}
	}

	cwd, err := os.Getwd()
	require.NoError(t, err, "Cannot Getwd")

	require.NoError(t, os.Chdir(path), "Cannot Chdir")

	*chdirSet = true
	return func() {
		err := os.Chdir(cwd)
		require.NoError(t, err, "Cannot Chdir in cleanup")

		*chdirSet = false
	}
}
