package gitlab

// Client is an internal HTTP client used for communication with GitLab
// instance
type Client interface {
	Resolve(domain string) *Lookup
}
