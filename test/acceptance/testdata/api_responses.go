package testdata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

type responseFn func(*testing.T, string) api.VirtualDomain

// DomainResponses holds the predefined API responses for certain domains
// that can be used with the GitLab API stub in acceptance tests
var DomainResponses = map[string]responseFn{
	"zip-from-disk.gitlab.io":           ZipFromFile,
	"zip-from-disk-not-found.gitlab.io": ZipFromFileNotFound,
	"zip-not-allowed-path.gitlab.io":    ZipFromNotAllowedPath,
	// test assume the working dir is inside shared/pages/
	"group.gitlab-example.com":        GenerateVirtualDomainFromDir("group", "group.gitlab-example.com"),
	"CapitalGroup.gitlab-example.com": GenerateVirtualDomainFromDir("CapitalGroup", "CapitalGroup.gitlab-example.com"),
	// NOTE: before adding more domains here, generate the zip archive by running (per project)
	// make zip PROJECT_SUBDIR=group/serving
	// make zip PROJECT_SUBDIR=group/project2
}

// ZipFromFile response for zip.gitlab.io
func ZipFromFile(t *testing.T, wd string) api.VirtualDomain {
	t.Helper()

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
					Type: "zip",
					Path: fmt.Sprintf("file://%s/@hashed/67/06/670671cd97404156226e507973f2ab8330d3022ca96e0c93bdbdb320c41adcaf/pages_deployments/01/artifacts.zip", wd),
				},
			},
		},
	}
}

// ZipFromFileNotFound response for zip-from-disk-not-found.gitlab.io
func ZipFromFileNotFound(t *testing.T, wd string) api.VirtualDomain {
	t.Helper()

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
					Type: "zip",
					Path: fmt.Sprintf("file://%s/@hashed/67/06/670671cd97404156226e507973f2ab8330d3022ca96e0c93bdbdb320c41adcaf/pages_deployments/01/unknown.zip", wd),
				},
			},
		},
	}
}

// ZipFromNotAllowedPath response for zip-not-allowed-path.gitlab.io
func ZipFromNotAllowedPath(t *testing.T, wd string) api.VirtualDomain {
	t.Helper()

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
					Type: "zip",
					// path outside of `pages-root`
					Path: "file:///some/random/path/public.zip",
				},
			},
		},
	}
}

// GenerateVirtualDomainFromDir walks the subdirectory inside of shared/pages/ to find any zip archives.
// It works for subdomains of pages-domain but not for custom domains (yet)
func GenerateVirtualDomainFromDir(dir, rootDomain string) responseFn {
	return func(t *testing.T, wd string) api.VirtualDomain {
		t.Helper()

		var foundZips []string

		// walk over dir and save any paths containing a `.zip` file
		// $(GITLAB_PAGES_DIR)/shared/pages + "/" + group

		cleanDir := filepath.Join(wd, dir)

		// make sure resolved path inside dir is under wd to avoid https://securego.io/docs/rules/g304.html
		require.Truef(t, strings.HasPrefix(cleanDir, wd), "path %q outside of wd %q", cleanDir, wd)

		filepath.Walk(cleanDir, func(path string, info os.FileInfo, err error) error {
			require.NoError(t, err)

			if strings.HasSuffix(info.Name(), ".zip") {
				project := strings.TrimPrefix(path, wd+"/"+dir)
				foundZips = append(foundZips, project)
			}

			return nil
		})

		lookupPaths := make([]api.LookupPath, 0, len(foundZips))
		// generate lookup paths
		for _, project := range foundZips {
			// if project = "group/subgroup/project/public.zip
			// trim prefix group and suffix /public.zip
			// so prefix = "/subgroup/project"
			prefix := strings.TrimPrefix(project, dir)
			prefix = strings.TrimSuffix(prefix, "/"+filepath.Base(project))

			// use / as prefix when the current prefix matches the rootDomain, e.g.
			// if request is group.gitlab-example.com/ and group/group.gitlab-example.com/public.zip exists
			if prefix == "/"+rootDomain {
				prefix = "/"
			}

			lookupPath := api.LookupPath{
				// TODO: find a way to configure this
				ProjectID:     123,
				AccessControl: false,
				HTTPSOnly:     false,
				Prefix:        prefix,
				Source: api.Source{
					Type: "zip",
					Path: fmt.Sprintf("file://%s", wd+"/"+dir+project),
				},
			}

			lookupPaths = append(lookupPaths, lookupPath)
		}

		return api.VirtualDomain{
			LookupPaths: lookupPaths,
		}
	}
}
