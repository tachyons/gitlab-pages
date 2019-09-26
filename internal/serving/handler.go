package serving

import "net/http"

// Handler interface represent an interface that is needed to fullfil the
// serving request
type Handler interface {
	Writer() http.ResponseWriter
	Request() *http.Request
	LookupPath() string
	Subpath() string
	IsNamespaceProject() bool
	IsHTTPSOnly() bool
	HasAccessControl() bool
	ProjectID() uint64
}
