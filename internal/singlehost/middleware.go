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
// which substitutes first path segment for host, e.g.:
// pages.example.com/group becames group.pages.example.com
// pages.example.com/group/subgroup/path/index.html becames group.pages.example.com/subgroup/path/index.html
func NewMiddleware(next http.Handler, pagesDomain string) http.Handler {
	return middleware{next: next, pagesDomain: pagesDomain}
}

func (m middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m.extractHostFromPath(r)

	m.next.ServeHTTP(w, r)
}

func (m middleware) extractHostFromPath(r *http.Request) {
	logger := log.WithFields(log.Fields{
		"orig_host":    r.Host,
		"orig_path":    r.URL.Path,
		"pages_domain": m.pagesDomain,
	})

	if !m.isTopPagesDomain(r.Host) {
		logger.Debug("Incoming request does not match pages domain")
		return
	}

	path := strings.TrimLeft(r.URL.Path, "/")
	segments := strings.SplitN(path, "/", 2)
	if len(segments) == 0 {
		logger.Debug("can't extract group from path because first segment is empty")
		return
	}

	namespace := segments[0]
	newPath := "/"

	if len(segments) > 1 {
		newPath += segments[1]
	}

	newHost := namespace + "." + r.Host

	logger.WithFields(log.Fields{
		"old_path": r.URL.Path,
		"new_path": newPath,
	}).Debug("Rewrite namespace host")

	r.Host = newHost
	r.URL.Path = newPath
}

func (m middleware) isTopPagesDomain(host string) bool {
	hostWithoutPort, _, err := net.SplitHostPort(host)
	if err != nil {
		hostWithoutPort = host
	}
	return hostWithoutPort == m.pagesDomain
}
