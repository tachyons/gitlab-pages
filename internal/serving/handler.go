package serving

import "net/http"

// Handler aggregates response/request and lookup path + subpath needed to
// handle a request and response.
type Handler struct {
	Writer     http.ResponseWriter
	Request    *http.Request
	LookupPath *LookupPath
	SubPath    string
}
