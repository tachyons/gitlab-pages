package factory

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

func TestLookupPath(t *testing.T) {
	t.Run("when lookup path is not a namespace project", func(t *testing.T) {
		lookup := api.LookupPath{Prefix: "/something"}

		path := LookupPath(1, lookup)

		require.Equal(t, path.Prefix, "/something")
		require.False(t, path.IsNamespaceProject)
	})

	t.Run("when lookup path is a namespace project", func(t *testing.T) {
		lookup := api.LookupPath{Prefix: "/"}

		path := LookupPath(2, lookup)

		require.Equal(t, path.Prefix, "/")
		require.True(t, path.IsNamespaceProject)
	})
}
