package gitlabstub

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

// APIResponses abstracts the stubbed responses for gitlabstub.
// The collection of responses is a map on the following format:
//
//	"domain": {
//		"prefix": {
//			ProjectID:     0,
//			AccessControl: false,
//			HTTPS:         false,
//			PathOnDisk:    "base/path/for/the/public.zip/file",
//		},
//	}
//
// For Example:
//
//	"group.gitlab-example.com": {
//		"/project1": {
//			ProjectID: 1000,
//			AccessControl: true,
//			HTTPS: true,
//			PathOnDisk: "group/project",
//		},
//		"/project2": {
//			ProjectID: 1001,
//			AccessControl: true,
//			HTTPS: true,
//			PathOnDisk: "group/project",
//		},
//	}
type APIResponses map[string]Responses

type Responses map[string]Response

// A Response is the struct responsible for creating a project LookupPath.
type Response struct {
	projectID     int
	accessControl bool
	httpsOnly     bool
	pathOnDisk    string // base directory is gitlab-pages/shared/pages
	uniqueHost    string
}

func (responses Responses) virtualDomain(wd string) api.VirtualDomain {
	return api.VirtualDomain{
		Certificate: "",
		Key:         "",
		LookupPaths: responses.lookupPaths(wd),
	}
}

func (responses Responses) lookupPaths(wd string) []api.LookupPath {
	lookupPaths := make([]api.LookupPath, 0, len(responses))

	for prefix, response := range responses {
		lookupPaths = append(lookupPaths, response.lookupPath(prefix, wd))
	}

	return lookupPaths
}

func (response Response) lookupPath(prefix, wd string) api.LookupPath {
	sourcePath := fmt.Sprintf("file://%s/%s/public.zip", wd, response.pathOnDisk)
	sum := sha256.Sum256([]byte(sourcePath))
	sha := hex.EncodeToString(sum[:])

	return api.LookupPath{
		Prefix:        prefix,
		ProjectID:     response.projectID,
		AccessControl: response.accessControl,
		HTTPSOnly:     response.httpsOnly,
		UniqueHost:    response.uniqueHost,
		Source: api.Source{
			Type:   "zip",
			Path:   sourcePath,
			SHA256: sha,
		},
	}
}

// apiResponses holds the predefined API responses for certain domains
// that can be used with the GitLab API stub in acceptance tests
var apiResponses = APIResponses{
	"zip-from-disk.gitlab.io": {
		"/": {
			projectID:  123,
			pathOnDisk: "@hashed/zip-from-disk.gitlab.io",
		},
	},
	"zip-not-allowed-path.gitlab.io": {
		"/": {
			pathOnDisk: "../../../../",
		},
	},
	"group.gitlab-example.com": {
		"/": {
			pathOnDisk: "group/group.gitlab-example.com",
		},
		"/CapitalProject": {
			pathOnDisk: "group/CapitalProject",
		},
		"/group.test.io": {
			pathOnDisk: "group/group.test.io",
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
		"/zip.gitlab.io": {
			pathOnDisk: "group/zip.gitlab.io",
		},
	},
	"group.404.gitlab-example.com": {
		"/": {
			pathOnDisk: "group.404/group.404.gitlab-example.com",
		},
		"/project.404": {
			pathOnDisk: "group.404/project.404",
		},
		"/project.no.404": {
			pathOnDisk: "group/project",
		},
		"/private_project": {
			pathOnDisk:    "group.404/private_project",
			projectID:     1300,
			accessControl: true,
		},
		"/private_unauthorized": {
			pathOnDisk:    "group.404/private_unauthorized",
			projectID:     2000,
			accessControl: true,
		},
	},
	"domain.404.com": {
		"/": {
			projectID:  1000,
			pathOnDisk: "group.404/domain.404",
		},
	},
	"CapitalGroup.gitlab-example.com": {
		"/CapitalProject": {
			pathOnDisk: "/CapitalGroup/CapitalProject",
		},
		"/project": {
			pathOnDisk: "/CapitalGroup/project",
		},
	},
	"group.https-only.gitlab-example.com": {
		"/project1": {
			projectID:  1000,
			httpsOnly:  true,
			pathOnDisk: "group.https-only/project1",
		},
		"/project2": {
			projectID:  1100,
			pathOnDisk: "group.https-only/project2",
		},
		"/project3": {
			pathOnDisk: "group.https-only/project3",
		},
		"/project4": {
			pathOnDisk: "group.https-only/project4",
		},
		"/project5": {
			pathOnDisk: "group.https-only/project5",
		},
	},
	"withacmechallenge.domain.com": {
		"/": {
			projectID:  1234,
			pathOnDisk: "group.acme/with.acme.challenge",
		},
	},
	"group.redirects.gitlab-example.com": {
		"/": {
			pathOnDisk: "group.redirects/group.redirects.gitlab-example.com",
		},
		"/custom-domain/": {
			pathOnDisk: "group.redirects/custom-domain",
		},
		"/project-redirects/": {
			pathOnDisk: "group.redirects/project-redirects",
		},
	},
	"redirects.custom-domain.com": {
		"/": {
			projectID:  1001,
			pathOnDisk: "group.redirects/custom-domain",
		},
	},
	"test.my-domain.com": {
		"/": {
			projectID:  1002,
			httpsOnly:  true,
			pathOnDisk: "group.https-only/project3",
		},
	},
	"test2.my-domain.com": {
		"/": {
			projectID:  1003,
			httpsOnly:  false,
			pathOnDisk: "group.https-only/project4",
		},
	},
	"no.cert.com": {
		"/": {
			projectID:  1004,
			httpsOnly:  true,
			pathOnDisk: "group.https-only/project5",
		},
	},
	"group.auth.gitlab-example.com": {
		"/": {
			projectID:     1005,
			accessControl: true,
			pathOnDisk:    "group.auth/group.auth.gitlab-example.com/",
		},
		"/private.project": {
			projectID:     1006,
			accessControl: true,
			pathOnDisk:    "group.auth/private.project/",
		},
		"/private.project.1": {
			projectID:     2006,
			accessControl: true,
			pathOnDisk:    "group.auth/private.project.1/",
		},
		"/private.project.2": {
			projectID:     3006,
			accessControl: true,
			pathOnDisk:    "group.auth/private.project.2/",
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
	},
	"private.domain.com": {
		"/": {
			projectID:     1007,
			accessControl: true,
			pathOnDisk:    "group.auth/private.project",
		},
	},
	"acmewithredirects.domain.com": {
		"/": {
			projectID:  1008,
			pathOnDisk: "group.acme/with.redirects",
		},
	},
	"group.unique-url.gitlab-example.com": {
		"/with-unique-url": {
			uniqueHost: "unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com",
			pathOnDisk: "group/project",
		},
		"/subgroup1/subgroup2/with-unique-url": {
			uniqueHost: "unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com",
			pathOnDisk: "group/project",
		},
		"/with-unique-url-with-port": {
			uniqueHost: "unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com",
			pathOnDisk: "group/project",
		},
		"/with-malformed-unique-url": {
			uniqueHost: "unique-url@gitlab-example.com:",
			pathOnDisk: "group/project",
		},
		"/with-different-protocol": {
			uniqueHost: "unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com",
			pathOnDisk: "group/project",
		},
		"/without-unique-url": {
			pathOnDisk: "group/project",
		},
	},
	"unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com": {
		"/": {
			uniqueHost: "unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com",
			pathOnDisk: "group/project",
		},
	},
	"unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com:8080": {
		"/": {
			uniqueHost: "unique-url-group-unique-url-a1b2c3d4e5f6.gitlab-example.com",
			pathOnDisk: "group/project",
		},
	},
	// NOTE: before adding more domains here, you can:
	// use an existing project or generate a new zip archive.
	// To generate a new zip archive run the following command, where PROJECT_SUBDIR
	// is a project folder within `gitlab-pages/shared/pages`
	// `make zip PROJECT_SUBDIR=group/serving`
}
