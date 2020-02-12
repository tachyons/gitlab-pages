package api

// Lookup defines an API lookup action with a response that GitLab sends
type Lookup struct {
	ETag   string
	Name   string
	Error  error
	Domain *VirtualDomain
}
