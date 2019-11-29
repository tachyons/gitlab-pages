package gitlab

// LookupPath represents a lookup path for  a GitLab Pages virtual domain
type LookupPath struct {
	ProjectID     int    `json:"project_id,omitempty"`
	AccessControl bool   `json:"access_control,omitempty"`
	HTTPSOnly     bool   `json:"https_only,omitempty"`
	Prefix        string `json:"prefix,omitempty"`
	Source        struct {
		Type string `json:"type,omitempty"`
		Path string `json:"path,omitempty"`
	}
}

// VirtualDomain represents a GitLab Pages virtual domain
type VirtualDomain struct {
	Certificate string `json:"certificate,omitempty"`
	Key         string `json:"key,omitempty"`

	LookupPaths []LookupPath `json:"lookup_paths"`
}
