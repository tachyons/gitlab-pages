package serving

// LookupPath holds a domain project configuration needed to handle a request
type LookupPath struct {
	Prefix             string // Project prefix, for example, /my/project in group.gitlab.io/my/project/index.html
	Path               string // Path is an internal and serving-specific location of a document
	IsNamespaceProject bool
	IsHTTPSOnly        bool
	HasAccessControl   bool
	ProjectID          uint64
}
