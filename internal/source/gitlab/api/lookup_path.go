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

// Serverless describeg serverless serving configuration
type Serverless struct {
	Function function `json:"function,omitempty"`
	Cluster  cluster  `json:"cluster,omitempty"`
}

type function struct {
	Name      string `json:"name,omitempty"`
	Domain    string `json:"domain,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type cluster struct {
	Address         string `json:"address,omitempty"`
	Port            string `json:"port,omitempty"`
	Hostname        string `json:"hostname,omitempty"`
	CertificateCert string `json:"cert,omitempty"`
	CertificateKey  string `json:"key,omitempty"`
}
