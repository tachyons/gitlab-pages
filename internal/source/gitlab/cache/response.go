package cache

// Response represents a response sent by GitLab domains source.
type Response struct {
	lookup *Lookup
	status int
	err    error
}

// Lookup is a helper method that returns all details of a Response
func (r *Response) Lookup() (*Lookup, int, error) {
	return r.lookup, r.status, r.err
}
