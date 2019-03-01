package client

var internalConfigs = map[string]DomainResponse{
	"group.internal.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:   "/project.internal/",
				DiskPath: "group.internal/project.internal/public",
			},
		},
	},
	"group.404.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:   "/project.no.404/",
				DiskPath: "group.404/project.no.404/public/",
			},
			LookupPath{
				Prefix:   "/project.404/",
				DiskPath: "group.404/project.404/public/",
			},
			LookupPath{
				Prefix:   "/project.404.symlink/",
				DiskPath: "group.404/project.404.symlink/public/",
			},
			LookupPath{
				Prefix:   "/domain.404/",
				DiskPath: "group.404/domain.404/public/",
			},
			LookupPath{
				Prefix:   "/group.404.test.io/",
				DiskPath: "group.404/group.404.test.io/public/",
			},
		},
	},
	"capitalgroup.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:   "/CapitalProject/",
				DiskPath: "CapitalGroup/CapitalProject/public/",
			},
			LookupPath{
				Prefix:   "/project/",
				DiskPath: "CapitalGroup/project/public/",
			},
		},
	},
	"group.auth.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:        "/private.project/",
				DiskPath:      "group.auth/private.project/public/",
				AccessControl: true,
				ProjectID:     1000,
			},
			LookupPath{
				Prefix:        "/private.project.1/",
				DiskPath:      "group.auth/private.project.1/public/",
				AccessControl: true,
				ProjectID:     2000,
			},
			LookupPath{
				Prefix:        "/private.project.2/",
				DiskPath:      "group.auth/private.project.2/public/",
				AccessControl: true,
				ProjectID:     3000,
			},
			LookupPath{
				Prefix:        "/subgroup/private.project/",
				DiskPath:      "group.auth/subgroup/private.project/public/",
				AccessControl: true,
				ProjectID:     1001,
			},
			LookupPath{
				Prefix:        "/subgroup/private.project.1/",
				DiskPath:      "group.auth/subgroup/private.project.1/public/",
				AccessControl: true,
				ProjectID:     2001,
			},
			LookupPath{
				Prefix:        "/subgroup/private.project.2/",
				DiskPath:      "group.auth/subgroup/private.project.2/public/",
				AccessControl: true,
				ProjectID:     3001,
			},
			LookupPath{
				Prefix:   "/group.auth.gitlab-example.com/",
				DiskPath: "group.auth/group.auth.gitlab-example.com/public/",
			},
			LookupPath{
				Prefix:           "/",
				DiskPath:         "group.auth/group.auth.gitlab-example.com/public/",
				NamespaceProject: true,
			},
		},
	},
	"group.https-only.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:    "/project5/",
				DiskPath:  "group.https-only/project5/public/",
				HTTPSOnly: true,
			},
			LookupPath{
				Prefix:   "/project4/",
				DiskPath: "group.https-only/project4/public/",
			},
			LookupPath{
				Prefix:   "/project3/",
				DiskPath: "group.https-only/project3/public/",
			},
			LookupPath{
				Prefix:   "/project2/",
				DiskPath: "group.https-only/project2/public/",
			},
			LookupPath{
				Prefix:    "/project1/",
				DiskPath:  "group.https-only/project1/public/",
				HTTPSOnly: true,
			},
			LookupPath{
				Prefix:           "/",
				DiskPath:         "group.auth/group.auth.gitlab-example.com/public/",
				NamespaceProject: true,
			},
		},
	},
	"group.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:   "/CapitalProject/",
				DiskPath: "group/CapitalProject/public/",
			},
			LookupPath{
				Prefix:   "/project/",
				DiskPath: "group/project/public/",
			},
			LookupPath{
				Prefix:   "/project2/",
				DiskPath: "group/project2/public/",
			},
			LookupPath{
				Prefix:   "/subgroup/project/",
				DiskPath: "group/subgroup/project/public/",
			},
			LookupPath{
				Prefix:   "/group.test.io/",
				DiskPath: "group/group.test.io/public/",
			},
			LookupPath{
				Prefix:           "/",
				DiskPath:         "group/group.gitlab-example.com/public/",
				NamespaceProject: true,
			},
		},
	},
	"nested.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:   "/sub1/sub2/sub3/sub4/sub5/project/",
				DiskPath: "nested/sub1/sub2/sub3/sub4/sub5/project/public/",
			},
			LookupPath{
				Prefix:   "/sub1/sub2/sub3/sub4/project/",
				DiskPath: "nested/sub1/sub2/sub3/sub4/project/public/",
			},
			LookupPath{
				Prefix:   "/sub1/sub2/sub3/project/",
				DiskPath: "nested/sub1/sub2/sub3/project/public/",
			},
			LookupPath{
				Prefix:   "/sub1/sub2/project/",
				DiskPath: "nested/sub1/sub2/project/public/",
			},
			LookupPath{
				Prefix:   "/sub1/project/",
				DiskPath: "nested/sub1/project/public/",
			},
			LookupPath{
				Prefix:   "/project/",
				DiskPath: "nested/project/public/",
			},
		},
	},

	// custom domains
	"domain.404.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:   "/",
				DiskPath: "group.404/domain.404.com/public/",
			},
		},
	},
	"private.domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:        "/",
				DiskPath:      "group.auth/private.project/public/",
				AccessControl: true,
				ProjectID:     1000,
			},
		},
	},
	"no.cert.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:    "/",
				DiskPath:  "group.https-only/project5/public/",
				HTTPSOnly: false,
			},
		},
	},
	"test2.my-domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:    "/",
				DiskPath:  "group.https-only/project4/public/",
				HTTPSOnly: false,
			},
		},
	},
	"test.my-domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:    "/",
				DiskPath:  "group.https-only/project3/public/",
				HTTPSOnly: true,
			},
		},
	},
	"test.domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:   "/",
				DiskPath: "group/group.test.io/public/",
			},
		},
	},
	"my.test.io": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:   "/",
				DiskPath: "group/group.test.io/public/",
			},
		},
	},
	"other.domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:   "/",
				DiskPath: "group/group.test.io/public/",
			},
		},
		Certificate: "test",
		Key:         "key",
	},
	"zip.domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:      "/",
				ArchivePath: "pages-deployment-100.zip",
			},
		},
	},
}

// MockRequestDomain provides a preconfigured set of domains
// for testing purposes
func MockRequestDomain(apiURL, host string) *DomainResponse {
	if response, ok := internalConfigs[host]; ok {
		return &response
	}

	return nil
}
