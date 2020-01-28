package serverless

import (
	"net/http"
	"strings"

	"github.com/tomasen/realip"
)

// NewDirectorFunc returns a director function capable of configuring a proxy
// request
func NewDirectorFunc(cluster Cluster) func(*http.Request) {
	return func(request *http.Request) {
		location := strings.Join([]string{cluster.Address, cluster.Port}, ":")

		request.Host = location
		request.URL.Host = location
		request.URL.Scheme = "https"
		request.Header.Set("User-Agent", "GitLab Pages Daemon")
		request.Header.Set("X-Forwarded-For", realip.FromRequest(request))
		request.Header.Set("X-Forwarded-Proto", "https")
	}
}
