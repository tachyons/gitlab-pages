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
	"group.gitlab-example.io": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/group.test.io/",
				Path:   "group/group.test.io/public/",
			},
			LookupPath{
				Prefix: "/",
				Path:   "group/group.gitlab-example.io/public/",
			},
		},
	},
	"group.test.io": DomainResponse{
		LookupPath: []LookupPath{
			LookupPath{
				Prefix: "/",
				Path:   "group/group.test.io/public/",
			},
		},
	},
}

func MockRequestDomain(apiUrl, host string) *DomainResponse {
	if response, ok := internalConfigs[host]; ok {
		println("Requested", host)
		return &response
	}
	return nil
}
