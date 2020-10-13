package serving

import "net/http"

// Request is a type that aggregates a serving itself, project lookup path and
// a request subpath based on an incoming request to serve page.
type Request struct {
	Serving    Serving     // Serving chosen to serve this request
	LookupPath *LookupPath // LookupPath contains pages project details
}

// ServeFileHTTP forwards serving request handler to the serving itself
func (s *Request) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	return s.Serving.ServeFileHTTP(w, r, s.LookupPath)
}

// ServeNotFoundHTTP forwards serving request handler to the serving itself
func (s *Request) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	s.Serving.ServeNotFoundHTTP(w, r, s.LookupPath)
}
