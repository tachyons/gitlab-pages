package serving

// Project holds a domain / project configuration
type LookupPath struct {
	Location           string
	Path               string
	IsNamespaceProject bool
	IsHTTPSOnly        bool
	HasAccessControl   bool
	ProjectID          uint64
}
