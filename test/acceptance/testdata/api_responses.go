package testdata

import (
	"fmt"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

type responseFn func(string) api.VirtualDomain

// DomainResponses holds the predefined API responses for certain domains
// that can be used with the GitLab API stub in acceptance tests
var DomainResponses = map[string]responseFn{
	"zip-from-disk.gitlab.io":           ZipFromFile,
	"zip-from-disk-not-found.gitlab.io": ZipFromFileNotFound,
}

// ZipFromFile response for zip.gitlab.io
func ZipFromFile(wd string) api.VirtualDomain {
	return api.VirtualDomain{
		Certificate: "",
		Key:         "",
		LookupPaths: []api.LookupPath{
			{
				ProjectID:     123,
				AccessControl: false,
				HTTPSOnly:     false,
				Prefix:        "/",
				Source: api.Source{
					Type:       "zip",
					Path:       fmt.Sprintf("file://%s/@hashed/67/06/670671cd97404156226e507973f2ab8330d3022ca96e0c93bdbdb320c41adcaf/pages_deployments/01/artifacts.zip", wd),
					Serverless: api.Serverless{},
				},
			},
		},
	}
}

// ZipFromFile response for zip.gitlab.io
func ZipFromFileNotFound(wd string) api.VirtualDomain {
	return api.VirtualDomain{
		Certificate: "",
		Key:         "",
		LookupPaths: []api.LookupPath{
			{
				ProjectID:     123,
				AccessControl: false,
				HTTPSOnly:     false,
				Prefix:        "/",
				Source: api.Source{
					Type:       "zip",
					Path:       fmt.Sprintf("file://%s/@hashed/67/06/670671cd97404156226e507973f2ab8330d3022ca96e0c93bdbdb320c41adcaf/pages_deployments/01/unknown.zip", wd),
					Serverless: api.Serverless{},
				},
			},
		},
	}
}
