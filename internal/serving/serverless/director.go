package serverless

import (
	"net/http"

	"github.com/tomasen/realip"
)

// NewDirectorFunc returns a director function capable of configuring a proxy
// request
func NewDirectorFunc(service string) func(*http.Request) {
	return func(request *http.Request) {
		request.Host = service
		request.URL.Host = service
		request.URL.Scheme = "https"
		request.Header.Set("User-Agent", "GitLab Pages Daemon")
		request.Header.Set("X-Forwarded-For", realip.FromRequest(request))
		request.Header.Set("X-Forwarded-Proto", "https")
	}
}
