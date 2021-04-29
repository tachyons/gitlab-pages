package gitlab

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
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
		require.EqualError(t, err, errDiskDisabled.Error())
		require.Nil(t, srv)
	})

	t.Run("when lookup path requires serverless serving", func(t *testing.T) {
		g := Gitlab{
			enableDisk: true,
		}

		lookup := api.LookupPath{
			Prefix: "/",
			Source: api.Source{
				Type: "serverless",
				Serverless: api.Serverless{
					Service: "my-func.knative.example.com",
					Cluster: api.Cluster{
						Address:         "127.0.0.10",
						Port:            "443",
						Hostname:        "my-cluster.example.com",
						CertificateCert: fixture.Certificate,
						CertificateKey:  fixture.Key,
					},
				},
			},
		}

		srv, err := g.fabricateServing(lookup)
		require.EqualError(t, err, fmt.Sprintf("gitlab: unkown serving source type: %q", lookup.Source.Type))

		// Serverless serving has been deprecated.
		// require.IsType(t, &serverless.Serverless{}, fabricateServing(lookup))
		require.Nil(t, srv)
	})
}
