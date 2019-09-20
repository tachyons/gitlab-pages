package domain

// Project represents GitLab project settings
type Project struct {
	NamespaceProject bool
	HTTPSOnly        bool
	AccessControl    bool
	ID               uint64
}
