package serverless

import "net/http"

// NewDirectorFunc returns a director function capable of configuring a proxy
// request
func NewDirectorFunc(cluster Cluster) func(*http.Request) {
	return func(r *http.Request) {
		// request.Host = domain
		// request.URL.Host = domain
		// request.URL.Scheme = "https"
		// request.Header.Set("User-Agent", "ReverseProxy PoC")
		// request.Header.Set("X-Forwarded ...")
	}
}
