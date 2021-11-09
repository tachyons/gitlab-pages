package serving

// LookupPath holds a domain project configuration needed to handle a request
type LookupPath struct {
	ServingType        string // Serving type being used, like `zip`
	Prefix             string // Project prefix, for example, /my/project in group.gitlab.io/my/project/index.html
	Path               string // Path is an internal and serving-specific location of a document
	SHA256             string
	IsNamespaceProject bool // IsNamespaceProject is DEPRECATED, see https://gitlab.com/gitlab-org/gitlab-pages/issues/272
	IsHTTPSOnly        bool
	HasAccessControl   bool
	ProjectID          uint64
}
