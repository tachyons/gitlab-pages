package domain

// ProjectConfig holds a custom project domain configuration
type ProjectConfig struct {
	DomainName    string
	Certificate   string
	Key           string
	HTTPSOnly     bool
	ProjectID     uint64
	AccessControl bool
}
