package domain

// VirtualDomain represents a GitLab Pages virtual domain that is being sent
// from GitLab
type VirtualDomain struct {
	Certificate string `json:"certificate,omitempty"`
	Key         string `json:"key,omitempty"`

	LookupPaths []LookupPath `json:"lookup_paths"`
}
