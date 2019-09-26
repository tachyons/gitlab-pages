package domain

// Project holds a domain / project configuration
type Project struct {
	LookupPath         string
	IsNamespaceProject bool
	IsHTTPSOnly        bool
	HasAccessControl   bool
	ID                 uint64
}
