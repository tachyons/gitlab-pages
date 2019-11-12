package cache

// Lookup defines a response that GitLab can send, which we can unmarshall
type Lookup struct {
	Domain          string
	CertificateCert string
	CertificateKey  string
	// TODO prefix hash map
	LookupPaths []struct {
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
