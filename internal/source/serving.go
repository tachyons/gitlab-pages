package source

import (
	"net/http"
)

// Serving represents a request and a response we use to serve pages for a
// given project and optional subpath in case of a domain with a subgroup
// access component
type Serving struct {
	Request *http.Request
	Writer  http.ResponseWriter
	Project string // project name
	SubPath string // request path
}

// TODO test, refactor related code
func (s *Serving) IsProjectFound() bool {
	return len(s.Project) > 0
}

func (s *Serving) requestPath() string {
	return s.Request.URL.Path
}
