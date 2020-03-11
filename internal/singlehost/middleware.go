package singlehost

import (
	"net"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
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
	host, port, err := net.SplitHostPort(r.Host)
	if err != nil {
		return
	}

	if host != m.pagesDomain {
		return
	}

	segments := strings.SplitN(r.URL.Path, "/", 3)
	namespace, newPath := segments[1], "/"+segments[2]

	newHost := namespace + "." + m.pagesDomain

	if port != "" {
		newHost += ":" + port
	}

	log.WithFields(log.Fields{
		"old_host": r.Host,
		"new_host": newHost,
		"old_path": r.URL.Path,
		"new_path": newPath,
	}).Debug("Rewrite namespace host")

	r.Host = newHost
	r.URL.Path = newPath
}
