package client

var internalConfigs = map[string]DomainResponse{
	"group.internal.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/project.internal/",
				Path:   "group.internal/project.internal/public",
			},
		},
	},
	"group.404.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/project.no.404/",
				Path:   "group.404/project.no.404/public/",
			},
			LookupPath{
				Prefix: "/project.404/",
				Path:   "group.404/project.404/public/",
			},
			LookupPath{
				Prefix: "/project.404.symlink/",
				Path:   "group.404/project.404.symlink/public/",
			},
			LookupPath{
				Prefix: "/domain.404/",
				Path:   "group.404/domain.404/public/",
			},
			LookupPath{
				Prefix: "/group.404.test.io/",
				Path:   "group.404/group.404.test.io/public/",
			},
		},
	},
	"capitalgroup.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/CapitalProject/",
				Path:   "CapitalGroup/CapitalProject/public/",
			},
			LookupPath{
				Prefix: "/project/",
				Path:   "CapitalGroup/project/public/",
			},
		},
	},
	"group.auth.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:        "/private.project/",
				Path:          "group.auth/private.project/public/",
				AccessControl: true,
				ProjectID:     1000,
			},
			LookupPath{
				Prefix:        "/private.project.1/",
				Path:          "group.auth/private.project.1/public/",
				AccessControl: true,
				ProjectID:     2000,
			},
			LookupPath{
				Prefix:        "/private.project.2/",
				Path:          "group.auth/private.project.2/public/",
				AccessControl: true,
				ProjectID:     3000,
			},
			LookupPath{
				Prefix:        "/subgroup/private.project/",
				Path:          "group.auth/subgroup/private.project/public/",
				AccessControl: true,
				ProjectID:     1001,
			},
			LookupPath{
				Prefix:        "/subgroup/private.project.1/",
				Path:          "group.auth/subgroup/private.project.1/public/",
				AccessControl: true,
				ProjectID:     2001,
			},
			LookupPath{
				Prefix:        "/subgroup/private.project.2/",
				Path:          "group.auth/subgroup/private.project.2/public/",
				AccessControl: true,
				ProjectID:     3001,
			},
			LookupPath{
				Prefix: "/group.auth.gitlab-example.com/",
				Path:   "group.auth/group.auth.gitlab-example.com/public/",
			},
			LookupPath{
				Prefix: "/",
				Path:   "group.auth/group.auth.gitlab-example.com/public/",
			},
		},
	},
	"group.https-only.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:    "/project5/",
				Path:      "group.https-only/project5/public/",
				HTTPSOnly: true,
			},
			LookupPath{
				Prefix: "/project4/",
				Path:   "group.https-only/project4/public/",
			},
			LookupPath{
				Prefix: "/project3/",
				Path:   "group.https-only/project3/public/",
			},
			LookupPath{
				Prefix: "/project2/",
				Path:   "group.https-only/project2/public/",
			},
			LookupPath{
				Prefix:    "/project1/",
				Path:      "group.https-only/project1/public/",
				HTTPSOnly: true,
			},
			LookupPath{
				Prefix: "/",
				Path:   "group.auth/group.auth.gitlab-example.com/public/",
			},
		},
	},
	"group.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/CapitalProject/",
				Path:   "group/CapitalProject/public/",
			},
			LookupPath{
				Prefix: "/project/",
				Path:   "group/project/public/",
			},
			LookupPath{
				Prefix: "/project2/",
				Path:   "group/project2/public/",
			},
			LookupPath{
				Prefix: "/subgroup/project/",
				Path:   "group/subgroup/project/public/",
			},
			LookupPath{
				Prefix: "/group.test.io/",
				Path:   "group/group.test.io/public/",
			},
			LookupPath{
				Prefix: "/",
				Path:   "group/group.gitlab-example.com/public/",
			},
		},
	},
	"nested.gitlab-example.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/sub1/sub2/sub3/sub4/sub5/project/",
				Path:   "nested/sub1/sub2/sub3/sub4/sub5/project/public/",
			},
			LookupPath{
				Prefix: "/sub1/sub2/sub3/sub4/project/",
				Path:   "nested/sub1/sub2/sub3/sub4/project/public/",
			},
			LookupPath{
				Prefix: "/sub1/sub2/sub3/project/",
				Path:   "nested/sub1/sub2/sub3/project/public/",
			},
			LookupPath{
				Prefix: "/sub1/sub2/project/",
				Path:   "nested/sub1/sub2/project/public/",
			},
			LookupPath{
				Prefix: "/sub1/project/",
				Path:   "nested/sub1/project/public/",
			},
			LookupPath{
				Prefix: "/project/",
				Path:   "nested/project/public/",
			},
		},
	},

	// custom domains
	"domain.404.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/",
				Path:   "group.404/domain.404.com/public/",
			},
		},
	},
	"private.domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:        "/",
				Path:          "group.auth/private.project/public/",
				AccessControl: true,
				ProjectID:     1000,
			},
		},
	},
	"no.cert.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:    "/",
				Path:      "group.https-only/project5/public/",
				HTTPSOnly: false,
			},
		},
	},
	"test2.my-domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:    "/",
				Path:      "group.https-only/project4/public/",
				HTTPSOnly: false,
			},
		},
	},
	"test.my-domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix:    "/",
				Path:      "group.https-only/project3/public/",
				HTTPSOnly: true,
			},
		},
	},
	"test.domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/",
				Path:   "group/group.test.io/public/",
			},
		},
	},
	"my.test.io": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/",
				Path:   "group/group.test.io/public/",
			},
		},
	},
	"other.domain.com": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/",
				Path:   "group/group.test.io/public/",
			},
		},
		Certificate: "test",
		Key:         "key",
	},
}

func MockRequestDomain(apiUrl, host string) *DomainResponse {
	if response, ok := internalConfigs[host]; ok {
		println("Requested", host)
		return &response
	}
	return nil
}
