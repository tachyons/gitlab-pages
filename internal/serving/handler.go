package serving

import "net/http"

// Handler agregates response/request and lookup path + subpath needed to
// handle a request and response.
type Handler struct {
	Writer     http.ResponseWriter
	Request    *http.Request
	LookupPath *LookupPath
	// Parsed representation of Request.URI that is part of LookupPath.Prefix
	SubPath string
}
