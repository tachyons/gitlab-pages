package serving

import "net/http"

// Serving is an interface used to define a serving driver
type Serving interface {
	ServeFileHTTP(w http.ResponseWriter, r *http.Request,
		lookupPath *LookupPath) bool
	ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request,
		lookupPath *LookupPath)
}
