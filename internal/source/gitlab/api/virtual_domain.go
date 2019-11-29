package api

// VirtualDomain represents a GitLab Pages virtual domain that is being sent
// from GitLab API
type VirtualDomain struct {
	Certificate string `json:"certificate,omitempty"`
	Key         string `json:"key,omitempty"`

	LookupPaths []LookupPath `json:"lookup_paths"`
}
