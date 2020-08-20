package serving

// LookupPath holds a domain project configuration needed to handle a request
type LookupPath struct {
	VFS                string // VFS to serve the given LookupPath for example: `local` or `zip`
	Prefix             string // Project prefix, for example, /my/project in group.gitlab.io/my/project/index.html
	Path               string // Path is an internal and serving-specific location of a document
	IsNamespaceProject bool   // IsNamespaceProject is DEPRECATED, see https://gitlab.com/gitlab-org/gitlab-pages/issues/272
	IsHTTPSOnly        bool
	HasAccessControl   bool
	ProjectID          uint64
}
