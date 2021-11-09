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
	Type   string `json:"type,omitempty"`
	Path   string `json:"path,omitempty"`
	SHA256 string `json:"sha256,omitempty"`
	Count  int    `json:"file_count,omitempty"`
	Size   int    `json:"file_size,omitempty"`
}
