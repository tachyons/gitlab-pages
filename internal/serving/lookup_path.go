package serving

// LookupPath holds a domain project configuration needed to handle a request
// TODO We might want to swap Location and Path names.
type LookupPath struct {
	Location           string // Location is a path to a project requested in a request
	Path               string // Path is an internal and serving-specific location of a document
	IsNamespaceProject bool
	IsHTTPSOnly        bool
	HasAccessControl   bool
	ProjectID          uint64
}
