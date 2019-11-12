package cache

// Lookup defines a response that GitLab sends
type Lookup struct {
	Domain Domain
	Status int
	Error  error
}

// Domain is a domain entry we store in cache
type Domain struct {
	Name            string
	CertificateCert string
	CertificateKey  string
	LookupPaths     map[string]struct {
		Prefix        string
		ProjectID     int
		HTTPSOnly     bool
		AccessControl bool
		Source        struct {
			Type string
			Path string
		}
	}
}
