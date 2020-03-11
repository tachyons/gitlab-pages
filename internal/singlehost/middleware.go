package singlehost

import (
	"net/http"
	"strings"
)

type middleware struct {
	next        http.Handler
	pagesDomain string
}

// NewMiddleware returns new single host middleware
// which substitutes first path segment for host
func NewMiddleware(next http.Handler, pagesDomain string) http.Handler {
	return middleware{next: next, pagesDomain: pagesDomain}
}

func (m middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.extractHostFromPath(r)

	m.next.ServeHTTP(w, r)
}

func (m middleware) extractHostFromPath(r *http.Request) {
	// custom domain
	if r.Host != m.pagesDomain {
		return
	}

	segments := strings.SplitN(r.URL.Path, "/", 2)
	namespace, newPath := segments[0], segments[1]

	r.Host = namespace + "." + m.pagesDomain
	r.URL.Path = newPath
}
