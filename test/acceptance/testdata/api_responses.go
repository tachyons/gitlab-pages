package testdata

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

type responseFn func(*testing.T, string) api.VirtualDomain

// DomainResponses holds the predefined API responses for certain domains
// that can be used with the GitLab API stub in acceptance tests
// Assume the working dir is inside shared/pages/
var DomainResponses = map[string]responseFn{
	"zip-from-disk.gitlab.io": customDomain(projectConfig{
		projectID:  123,
		pathOnDisk: "@hashed/zip-from-disk.gitlab.io",
	}, true),
	"zip-from-disk-not-found.gitlab.io": customDomain(projectConfig{}, true),
	// outside of working dir
	"zip-not-allowed-path.gitlab.io":  customDomain(projectConfig{pathOnDisk: "../../../../"}, true),
	"group.gitlab-example.com":        generateVirtualDomainFromDir("group", "group.gitlab-example.com", nil),
	"CapitalGroup.gitlab-example.com": generateVirtualDomainFromDir("CapitalGroup", "CapitalGroup.gitlab-example.com", nil),
	"group.404.gitlab-example.com":    generateVirtualDomainFromDir("group.404", "group.404.gitlab-example.com", nil),
	"group.https-only.gitlab-example.com": generateVirtualDomainFromDir("group.https-only", "group.https-only.gitlab-example.com", map[string]projectConfig{
		"/project1": {
			projectID: 1000,
			https:     true,
		},
		"/project2": {
			projectID: 1100,
			https:     false,
		},
	}),
	"domain.404.com": customDomain(projectConfig{
		projectID:  1000,
		pathOnDisk: "group.404/domain.404",
	}, false),
	"withacmechallenge.domain.com": customDomain(projectConfig{
		projectID:  1234,
		pathOnDisk: "group.acme/with.acme.challenge",
	}, false),
	// NOTE: before adding more domains here, generate the zip archive by running (per project)
	// make zip PROJECT_SUBDIR=group/serving
	// make zip PROJECT_SUBDIR=group/project2
}

// generateVirtualDomainFromDir walks the subdirectory inside of shared/pages/ to find any zip archives.
// It works for subdomains of pages-domain but not for custom domains (yet)
func generateVirtualDomainFromDir(dir, rootDomain string, perPrefixConfig map[string]projectConfig) responseFn {
	return func(t *testing.T, wd string) api.VirtualDomain {
		t.Helper()

		var foundZips []string
		handlerPaths := map[string]string{}
		//path   "group.https-only.gitlab-example.com/project2/public.zip"
		//handler"group.https-only.gitlab-example.com/project2/public.zip"
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
				handlerPaths[strings.ToLower(project)] = path
			}

			return nil
		})

		testServerURL := newZipFileServer(t, handlerPaths)

		lookupPaths := make([]api.LookupPath, 0, len(foundZips))
		// generate lookup paths
		for _, project := range foundZips {
			// if project = "group/subgroup/project/public.zip
			// trim prefix group and suffix /public.zip
			// so prefix = "/subgroup/project"
			prefix := strings.TrimPrefix(project, dir)
			prefix = strings.TrimSuffix(prefix, "/"+filepath.Base(project))

			//urlPath :=
			// use / as prefix when the current prefix matches the rootDomain, e.g.
			// if request is group.gitlab-example.com/ and group/group.gitlab-example.com/public.zip exists
			if prefix == "/"+rootDomain {
				prefix = "/"
				// so it can be found by the testServerURL handler
				//urlPath = project + "/public.zip"
			}

			cfg, ok := perPrefixConfig[prefix]
			if !ok {
				cfg = projectConfig{}
			}

			lookupPath := api.LookupPath{
				ProjectID:     cfg.projectID,
				AccessControl: cfg.accessControl,
				HTTPSOnly:     cfg.https,
				Prefix:        prefix,
				Source: api.Source{
					Type: "zip",
					Path: fmt.Sprintf("%s%s", testServerURL, strings.ToLower(project)),
				},
			}

			lookupPaths = append(lookupPaths, lookupPath)
		}

		return api.VirtualDomain{
			LookupPaths: lookupPaths,
		}
	}
}

type projectConfig struct {
	// refer to makeGitLabPagesAccessStub for custom HTTP responses per projectID
	projectID     int
	accessControl bool
	https         bool
	pathOnDisk    string
}

// customDomain with per project config
func customDomain(config projectConfig, serveFromDisk bool) responseFn {
	return func(t *testing.T, wd string) api.VirtualDomain {
		t.Helper()

		path := fmt.Sprintf("file://%s/%s/public.zip", wd, config.pathOnDisk)
		if !serveFromDisk {
			cleanPath := filepath.Clean(wd + "/" + config.pathOnDisk + "/public.zip")
			testServerURL := newZipFileServer(t, map[string]string{
				"/" + config.pathOnDisk + "/public.zip": cleanPath,
			})
			path = fmt.Sprintf("%s/%s/public.zip", testServerURL, config.pathOnDisk)
		}

		return api.VirtualDomain{
			Certificate: "",
			Key:         "",
			LookupPaths: []api.LookupPath{
				{
					ProjectID:     config.projectID,
					AccessControl: config.accessControl,
					HTTPSOnly:     config.https,
					// prefix should always be `/` for custom domains, otherwise `resolvePath` will try
					// to look for files under public/prefix/ when serving content instead of just public/
					// see internal/serving/disk/ for details
					Prefix: "/",
					Source: api.Source{
						Type: "zip",
						Path: path,
					},
				},
			},
		}
	}
}

func newZipFileServer(t *testing.T, projectPaths map[string]string) string {
	t.Helper()

	mux := http.NewServeMux()
	for urlPath, diskPath := range projectPaths {
		fmt.Printf("creating handler for: %q in - %q\n\n", urlPath, diskPath)
		mux.HandleFunc(urlPath, func(w http.ResponseWriter, r *http.Request) {
			fmt.Printf("FULL REQ URL: %q\n", r.URL.Path)
			fmt.Printf("INSIDE THE HANDLER FOR: %q - opening:%q\n\n", urlPath, diskPath)
			fi, err := os.Lstat(diskPath)
			require.NoError(t, err)
			require.False(t, fi.IsDir())
			http.ServeFile(w, r, diskPath)
		})
	}
	v := reflect.ValueOf(mux).Elem()
	fmt.Printf("routes: %v\n", v.FieldByName("m"))
	testServer := httptest.NewServer(mux)

	t.Cleanup(func() {
		testServer.Close()
	})

	return testServer.URL
}
