package serving

import "net/http"

// Request is a type that aggregates a serving itself, project lookup path and
// a request subpath based on an incoming request to serve page.
type Request struct {
	Serving    Serving     // Serving chosen to serve this request
	LookupPath *LookupPath // LookupPath contains pages project details
	SubPath    string      // Subpath is a URL path subcomponent for this request
}

// ServeFileHTTP forwards serving request handler to the serving itself
func (s *Request) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	handler := Handler{
		Writer:     w,
		Request:    r,
		LookupPath: s.LookupPath,
		SubPath:    s.SubPath,
	}

	return s.Serving.ServeFileHTTP(handler)
}

// ServeNotFoundHTTP forwards serving request handler to the serving itself
func (s *Request) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	handler := Handler{
		Writer:     w,
		Request:    r,
		LookupPath: s.LookupPath,
		SubPath:    s.SubPath,
	}

	s.Serving.ServeNotFoundHTTP(handler)
}
