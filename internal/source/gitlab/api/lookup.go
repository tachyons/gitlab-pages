package api

// Lookup defines an API lookup action with a response that GitLab sends
type Lookup struct {
	Name   string
	Domain VirtualDomain
	Status int
	Error  error
}
