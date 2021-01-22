package testdata

import (
	"fmt"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

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
