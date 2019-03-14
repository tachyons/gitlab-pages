package testdata

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/client"
)

var internalConfigs = map[string]client.DomainResponse{
	"group.internal.gitlab-example.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix: "/project.internal/",
				Path:   "group.internal/project.internal/public",
			},
		},
	},
	"group.404.gitlab-example.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix: "/project.no.404/",
				Path:   "group.404/project.no.404/public/",
			},
			client.LookupPath{
				Prefix: "/project.404/",
				Path:   "group.404/project.404/public/",
			},
			client.LookupPath{
				Prefix: "/project.404.symlink/",
				Path:   "group.404/project.404.symlink/public/",
			},
			client.LookupPath{
				Prefix: "/domain.404/",
				Path:   "group.404/domain.404/public/",
			},
			client.LookupPath{
				Prefix: "/group.404.test.io/",
				Path:   "group.404/group.404.test.io/public/",
			},
		},
	},
	"capitalgroup.gitlab-example.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix: "/CapitalProject/",
				Path:   "CapitalGroup/CapitalProject/public/",
			},
			client.LookupPath{
				Prefix: "/project/",
				Path:   "CapitalGroup/project/public/",
			},
		},
	},
	"group.auth.gitlab-example.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix:        "/private.project/",
				Path:          "group.auth/private.project/public/",
				AccessControl: true,
				ProjectID:     1000,
			},
			client.LookupPath{
				Prefix:        "/private.project.1/",
				Path:          "group.auth/private.project.1/public/",
				AccessControl: true,
				ProjectID:     2000,
			},
			client.LookupPath{
				Prefix:        "/private.project.2/",
				Path:          "group.auth/private.project.2/public/",
				AccessControl: true,
				ProjectID:     3000,
			},
			client.LookupPath{
				Prefix:        "/subgroup/private.project/",
				Path:          "group.auth/subgroup/private.project/public/",
				AccessControl: true,
				ProjectID:     1001,
			},
			client.LookupPath{
				Prefix:        "/subgroup/private.project.1/",
				Path:          "group.auth/subgroup/private.project.1/public/",
				AccessControl: true,
				ProjectID:     2001,
			},
			client.LookupPath{
				Prefix:        "/subgroup/private.project.2/",
				Path:          "group.auth/subgroup/private.project.2/public/",
				AccessControl: true,
				ProjectID:     3001,
			},
			client.LookupPath{
				Prefix: "/group.auth.gitlab-example.com/",
				Path:   "group.auth/group.auth.gitlab-example.com/public/",
			},
			client.LookupPath{
				Prefix:           "/",
				Path:             "group.auth/group.auth.gitlab-example.com/public/",
				NamespaceProject: true,
			},
		},
	},
	"group.https-only.gitlab-example.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix:    "/project5/",
				Path:      "group.https-only/project5/public/",
				HTTPSOnly: true,
			},
			client.LookupPath{
				Prefix: "/project4/",
				Path:   "group.https-only/project4/public/",
			},
			client.LookupPath{
				Prefix: "/project3/",
				Path:   "group.https-only/project3/public/",
			},
			client.LookupPath{
				Prefix: "/project2/",
				Path:   "group.https-only/project2/public/",
			},
			client.LookupPath{
				Prefix:    "/project1/",
				Path:      "group.https-only/project1/public/",
				HTTPSOnly: true,
			},
			client.LookupPath{
				Prefix:           "/",
				Path:             "group.auth/group.auth.gitlab-example.com/public/",
				NamespaceProject: true,
			},
		},
	},
	"group.gitlab-example.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix: "/CapitalProject/",
				Path:   "group/CapitalProject/public/",
			},
			client.LookupPath{
				Prefix: "/project/",
				Path:   "group/project/public/",
			},
			client.LookupPath{
				Prefix: "/project2/",
				Path:   "group/project2/public/",
			},
			client.LookupPath{
				Prefix: "/subgroup/project/",
				Path:   "group/subgroup/project/public/",
			},
			client.LookupPath{
				Prefix: "/group.test.io/",
				Path:   "group/group.test.io/public/",
			},
			client.LookupPath{
				Prefix:           "/",
				Path:             "group/group.gitlab-example.com/public/",
				NamespaceProject: true,
			},
		},
	},
	"nested.gitlab-example.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix: "/sub1/sub2/sub3/sub4/sub5/project/",
				Path:   "nested/sub1/sub2/sub3/sub4/sub5/project/public/",
			},
			client.LookupPath{
				Prefix: "/sub1/sub2/sub3/sub4/project/",
				Path:   "nested/sub1/sub2/sub3/sub4/project/public/",
			},
			client.LookupPath{
				Prefix: "/sub1/sub2/sub3/project/",
				Path:   "nested/sub1/sub2/sub3/project/public/",
			},
			client.LookupPath{
				Prefix: "/sub1/sub2/project/",
				Path:   "nested/sub1/sub2/project/public/",
			},
			client.LookupPath{
				Prefix: "/sub1/project/",
				Path:   "nested/sub1/project/public/",
			},
			client.LookupPath{
				Prefix: "/project/",
				Path:   "nested/project/public/",
			},
		},
	},

	// custom domains
	"domain.404.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix: "/",
				Path:   "group.404/domain.404.com/public/",
			},
		},
	},
	"private.domain.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix:        "/",
				Path:          "group.auth/private.project/public/",
				AccessControl: true,
				ProjectID:     1000,
			},
		},
	},
	"no.cert.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix:    "/",
				Path:      "group.https-only/project5/public/",
				HTTPSOnly: false,
			},
		},
	},
	"test2.my-domain.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix:    "/",
				Path:      "group.https-only/project4/public/",
				HTTPSOnly: false,
			},
		},
	},
	"test.my-domain.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix:    "/",
				Path:      "group.https-only/project3/public/",
				HTTPSOnly: true,
			},
		},
	},
	"test.domain.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix: "/",
				Path:   "group/group.test.io/public/",
			},
		},
	},
	"my.test.io": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix: "/",
				Path:   "group/group.test.io/public/",
			},
		},
	},
	"other.domain.com": client.DomainResponse{
		LookupPath: []client.LookupPath{
			client.LookupPath{
				Prefix: "/",
				Path:   "group/group.test.io/public/",
			},
		},
		Certificate: "test",
		Key:         "key",
	},
}
