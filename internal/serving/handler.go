package serving

import "net/http"

type Handler struct {
	Writer  http.ResponseWriter
	Request *http.Request

	LookupPath *LookupPath

	// parsed representation of Request.URI
	// that is part of LookupPath.Prefix
	SubPath string
}
