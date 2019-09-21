package serving

import "net/http"

// Serving represents an interface used to serve pages for a given domain /
// address
type Serving interface {
	ServeHTTP(http.ResponseWriter, *http.Request)
}
