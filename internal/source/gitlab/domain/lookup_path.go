package domain

// LookupPath represents a lookup path for a virtual domain
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
