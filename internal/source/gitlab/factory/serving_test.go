package factory

import (
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/fixture"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/serverless"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

func TestServing(t *testing.T) {
	t.Run("when lookup path requires disk serving", func(t *testing.T) {
		lookup := api.LookupPath{
			Prefix: "/",
			Source: api.Source{Type: "file"},
		}

		require.IsType(t, &disk.Disk{}, Serving(lookup))
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

		require.IsType(t, &serverless.Serverless{}, Serving(lookup))
	})
}
