package serving

// LookupPath holds a domain project configuration needed to handle a request
type LookupPath struct {
	Location           string
	Path               string
	IsNamespaceProject bool
	IsHTTPSOnly        bool
	HasAccessControl   bool
	ProjectID          uint64
}
