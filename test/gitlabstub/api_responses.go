package gitlabstub

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

type responseFn func(string) api.VirtualDomain

type projectConfig struct {
	// refer to makeGitLabPagesAccessStub for custom HTTP responses per projectID
	projectID     int
	accessControl bool
	https         bool
	pathOnDisk    string
}

// domainResponses holds the predefined API responses for certain domains
// that can be used with the GitLab API stub in acceptance tests
// Assume the working dir is inside shared/pages/
var domainResponses = map[string]responseFn{
	"zip-from-disk.gitlab.io": customDomain(projectConfig{
		projectID:  123,
		pathOnDisk: "@hashed/zip-from-disk.gitlab.io",
	}),
	"zip-from-disk-not-found.gitlab.io": customDomain(projectConfig{}),
	"zip-not-allowed-path.gitlab.io":    customDomain(projectConfig{pathOnDisk: "../../../../"}),
	"group.gitlab-example.com": generateVirtualDomain("group.gitlab-example.com", map[string]projectConfig{
		"/CapitalProject": {
			pathOnDisk: "group/CapitalProject",
		},
		"/group.gitlab-example.com": {
			pathOnDisk: "group/group.gitlab-example.com",
		},
		"/new-source-test.gitlab.io": {
			pathOnDisk: "group/new-source-test.gitlab.io",
		},
		"/project": {
			pathOnDisk: "group/project",
		},
		"/project2": {
			pathOnDisk: "group/project2",
		},
		"/serving": {
			pathOnDisk: "group/serving",
		},
		"/subgroup/project": {
			pathOnDisk: "group/subgroup/project",
		},
		"/zip.gitlab.io/public-without-dirs": {
			pathOnDisk: "group/zip.gitlab.io",
		},
		"/zip.gitlab.io": {
			pathOnDisk: "group/zip.gitlab.io",
		},
	}),
	"CapitalGroup.gitlab-example.com": generateVirtualDomain("CapitalGroup.gitlab-example.com", map[string]projectConfig{
		"/CapitalProject": {
			pathOnDisk: "CapitalProject/CapitalProject",
		},
	}),
	"group.404.gitlab-example.com": generateVirtualDomain("group.404.gitlab-example.com", map[string]projectConfig{
		"/private_project": {
			projectID:     1300,
			accessControl: true,
			pathOnDisk:    "group.404/private_project",
		},
		"/private_unauthorized": {
			projectID:     2000,
			accessControl: true,
			pathOnDisk:    "group.404/private_unauthorized",
		},
	}),
	"group.https-only.gitlab-example.com": generateVirtualDomain("group.https-only.gitlab-example.com", map[string]projectConfig{
		"/project1": {
			projectID:  1000,
			https:      true,
			pathOnDisk: "group.https-only/project1",
		},
		"/project2": {
			projectID:  1100,
			https:      false,
			pathOnDisk: "group.https-only/project2",
		},
	}),
	"domain.404.com": customDomain(projectConfig{
		projectID:  1000,
		pathOnDisk: "group.404/domain.404",
	}),
	"withacmechallenge.domain.com": customDomain(projectConfig{
		projectID:  1234,
		pathOnDisk: "group.acme/with.acme.challenge",
	}),
	"group.redirects.gitlab-example.com": generateVirtualDomain("group.redirects.gitlab-example.com", map[string]projectConfig{
		"/custom-domain": {
			pathOnDisk: "group.redirects/custom-domain",
		},
		"/group.redirects.gitlab-example.com": {
			pathOnDisk: "group.redirects/group.redirects.gitlab-example.com",
		},
		"/project-redirects": {
			pathOnDisk: "group.redirects/project-redirects",
		},
	}),
	"redirects.custom-domain.com": customDomain(projectConfig{
		projectID:  1001,
		pathOnDisk: "group.redirects/custom-domain",
	}),
	"test.my-domain.com": customDomain(projectConfig{
		projectID:  1002,
		https:      true,
		pathOnDisk: "group.https-only/project3",
	}),
	"test2.my-domain.com": customDomain(projectConfig{
		projectID:  1003,
		https:      false,
		pathOnDisk: "group.https-only/project4",
	}),
	"no.cert.com": customDomain(projectConfig{
		projectID:  1004,
		https:      true,
		pathOnDisk: "group.https-only/project5",
	}),
	"group.auth.gitlab-example.com": generateVirtualDomain("group.auth.gitlab-example.com", map[string]projectConfig{
		"/": {
			projectID:     1005,
			accessControl: true,
			pathOnDisk:    "group.auth/group.auth.gitlab-example.com",
		},
		"/private.project": {
			projectID:     1006,
			accessControl: true,
			pathOnDisk:    "group.auth/private.project",
		},
		"/private.project.1": {
			projectID:     2006,
			accessControl: true,
			pathOnDisk:    "group.auth/private.project.1",
		},
		"/private.project.2": {
			projectID:     3006,
			accessControl: true,
			pathOnDisk:    "group.auth/private.project.2",
		},
		"/subgroup/private.project": {
			projectID:     1007,
			accessControl: true,
			pathOnDisk:    "group.auth/subgroup/private.project",
		},
		"/subgroup/private.project.1": {
			projectID:     2007,
			accessControl: true,
			pathOnDisk:    "group.auth/subgroup/private.project.1",
		},
		"/subgroup/private.project.2": {
			projectID:     3007,
			accessControl: true,
			pathOnDisk:    "group.auth/subgroup/private.project.2",
		},
	}),
	"private.domain.com": customDomain(projectConfig{
		projectID:     1007,
		accessControl: true,
		pathOnDisk:    "group.auth/private.project",
	}),
	"acmewithredirects.domain.com": customDomain(projectConfig{
		projectID:  1008,
		pathOnDisk: "group.acme/with.redirects",
	}),
	// NOTE: before adding more domains here, generate the zip archive by running (per project)
	// make zip PROJECT_SUBDIR=group/serving
	// make zip PROJECT_SUBDIR=group/project2
}

func generateVirtualDomain(rootDomain string, projectConfigs map[string]projectConfig) responseFn {
	return func(wd string) api.VirtualDomain {
		nextID := 1000
		lookupPaths := make([]api.LookupPath, 0, len(projectConfigs))

		for project, config := range projectConfigs {
			if config.projectID == 0 {
				config.projectID = nextID
				nextID++
			}

			sourcePath := fmt.Sprintf("file://%s", filepath.Join(wd, config.pathOnDisk, "public.zip"))

			sum := sha256.Sum256([]byte(sourcePath))
			sha := hex.EncodeToString(sum[:])

			lookupPaths = append(lookupPaths, api.LookupPath{
				ProjectID:     config.projectID,
				AccessControl: config.accessControl,
				HTTPSOnly:     config.https,
				Prefix:        ensureEndingSlash(project),
				Source: api.Source{
					Type:   "zip",
					Path:   sourcePath,
					SHA256: sha,
				},
			})
		}

		return api.VirtualDomain{
			Certificate: "",
			Key:         "",
			LookupPaths: lookupPaths,
		}
	}
}

// generateVirtualDomainFromDir walks the subdirectory inside of shared/pages/ to find any zip archives.
// It works for subdomains of pages-domain but not for custom domains (yet)
func generateVirtualDomainFromDir(dir, rootDomain string, perPrefixConfig map[string]projectConfig) responseFn {
	return func(wd string) api.VirtualDomain {
		var foundZips []string

		// walk over dir and save any paths containing a `.zip` file
		// $(GITLAB_PAGES_DIR)/shared/pages + "/" + group

		cleanDir := filepath.Join(wd, dir)

		// make sure resolved path inside dir is under wd to avoid https://securego.io/docs/rules/g304.html
		if !strings.HasPrefix(cleanDir, wd) {
			log.Fatalf("path %q outside of wd %q", cleanDir, wd)
		}

		walkErr := filepath.Walk(cleanDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if strings.HasSuffix(info.Name(), ".zip") {
				project := strings.TrimPrefix(path, wd+"/"+dir)
				foundZips = append(foundZips, project)
			}

			return nil
		})

		if walkErr != nil {
			log.Fatal(walkErr)
		}

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

			cfg, ok := perPrefixConfig[prefix]
			if !ok {
				cfg = projectConfig{}
			}

			sourcePath := fmt.Sprintf("file://%s", wd+"/"+dir+project)
			sum := sha256.Sum256([]byte(sourcePath))
			sha := hex.EncodeToString(sum[:])

			lookupPath := api.LookupPath{
				ProjectID:     cfg.projectID,
				AccessControl: cfg.accessControl,
				HTTPSOnly:     cfg.https,
				// gitlab.Resolve logic expects prefix to have ending slash
				Prefix: ensureEndingSlash(prefix),
				Source: api.Source{
					Type:   "zip",
					Path:   sourcePath,
					SHA256: sha,
				},
			}

			lookupPaths = append(lookupPaths, lookupPath)
		}

		return api.VirtualDomain{
			LookupPaths: lookupPaths,
		}
	}
}

// customDomain with per project config
func customDomain(config projectConfig) responseFn {
	return func(wd string) api.VirtualDomain {
		sourcePath := fmt.Sprintf("file://%s/%s/public.zip", wd, config.pathOnDisk)
		sum := sha256.Sum256([]byte(sourcePath))
		sha := hex.EncodeToString(sum[:])

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
						Type:   "zip",
						SHA256: sha,
						Path:   sourcePath,
					},
				},
			},
		}
	}
}

func ensureEndingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}

	return path + "/"
}
