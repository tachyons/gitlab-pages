package gitlab

// Lookup defines a response that GitLab can send, which we can unmarshall
type Lookup struct {
	Domain          string
	CertificateCert string
	CertificateKey  string
	Serving         string
	Prefix          string
	LookupPaths     []struct {
		ProjectID     int
		HTTPSOnly     bool
		AccessControl bool
		Source        struct {
			Type string
			Path string
		}
	}
}
