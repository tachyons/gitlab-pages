package gitlab

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/serverless"
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
		lookup := api.LookupPath{
			Prefix: "/",
			Source: api.Source{Type: "file"},
		}

		require.IsType(t, &disk.Disk{}, fabricateServing(lookup))
	})

	t.Run("when lookup path requires serverless serving", func(t *testing.T) {
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

		require.IsType(t, &serverless.Serverless{}, fabricateServing(lookup))
	})
}
