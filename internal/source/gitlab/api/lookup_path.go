package api

// LookupPath represents a lookup path for a virtual domain
type LookupPath struct {
	ProjectID     int    `json:"project_id,omitempty"`
	AccessControl bool   `json:"access_control,omitempty"`
	HTTPSOnly     bool   `json:"https_only,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
	Source        Source `json:"source,omitempty"`
}

// Source describes GitLab Page serving variant
type Source struct {
	Type       string     `json:"type,omitempty"`
	Path       string     `json:"path,omitempty"`
	Serverless Serverless `json:"serverless,omitempty"`
}

// Serverless describes serverless serving configuration
type Serverless struct {
	Service string  `json:"service,omitempty"`
	Cluster Cluster `json:"cluster,omitempty"`
}

// Cluster describes serverless cluster configuration
type Cluster struct {
	Address         string `json:"address,omitempty"`
	Port            string `json:"port,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
	CertificateCert string `json:"cert,omitempty"`
	CertificateKey  string `json:"key,omitempty"`
}
