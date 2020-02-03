package serving

import "net/http"

// Handler aggregates response/request and lookup path + subpath needed to
// handle a request and response.
type Handler struct {
	Writer     http.ResponseWriter
	Request    *http.Request
	LookupPath *LookupPath
	Serving    Serving
	SubPath    string
}

// ServeFileHTTP passes the handler itself to a serving function
func (h Handler) ServeFileHTTP() bool {
	return h.Serving.ServeFileHTTP(h)
}

// ServeNotFoundHTTP passes the handler itself to a serving function
func (h Handler) ServeNotFoundHTTP() {
	h.Serving.ServeNotFoundHTTP(h)
}
