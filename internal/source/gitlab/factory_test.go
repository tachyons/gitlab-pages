package gitlab

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

func TestFabricateLookupPath(t *testing.T) {
	t.Run("when lookup path is not a namespace project", func(t *testing.T) {
		lookup := api.LookupPath{Prefix: "/something"}

		path := fabricateLookupPath(1, lookup)

		require.Equal(t, path.Prefix, "/something")
		require.False(t, path.IsNamespaceProject)
	})

	t.Run("when lookup path is a namespace project", func(t *testing.T) {
		lookup := api.LookupPath{Prefix: "/"}

		path := fabricateLookupPath(2, lookup)

		require.Equal(t, path.Prefix, "/")
		require.True(t, path.IsNamespaceProject)
	})
}

func TestFabricateServing(t *testing.T) {
	t.Run("when lookup path requires disk serving", func(t *testing.T) {
		g := Gitlab{
			enableDisk: true,
		}

		lookup := api.LookupPath{
			Prefix: "/",
			Source: api.Source{Type: "file"},
		}
		srv, err := g.fabricateServing(lookup)
		require.NoError(t, err)
		require.IsType(t, &disk.Disk{}, srv)
	})

	t.Run("when lookup path requires disk serving but disk is disabled", func(t *testing.T) {
		g := Gitlab{
			enableDisk: false,
		}

		lookup := api.LookupPath{
			Prefix: "/",
			Source: api.Source{Type: "file"},
		}
		srv, err := g.fabricateServing(lookup)
		require.ErrorIs(t, err, ErrDiskDisabled)
		require.Nil(t, srv)
	})
}
