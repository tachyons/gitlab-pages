package serverless

import (
	"net/http"
	"strings"
)

// NewDirectorFunc returns a director function capable of configuring a proxy
// request
func NewDirectorFunc(cluster Cluster) func(*http.Request) {
	return func(request *http.Request) {
		request.Host = cluster.Address
		request.URL.Host = strings.Join([]string{cluster.Address, cluster.Port}, ":")
		request.URL.Scheme = "https"
		request.Header.Set("User-Agent", "GitLab Pages Daemon")
		request.Header.Set("X-Forwarded-For", "123") // TODO
	}
}
