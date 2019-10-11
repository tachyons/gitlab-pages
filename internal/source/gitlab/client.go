package gitlab

// client is an internal HTTP client used for communication with GitLab
// instance
type client interface {
	Resolve(domain string) *Lookup
}
