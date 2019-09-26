package disk

import "net/http"

type handler interface {
	Writer() http.ResponseWriter
	Request() *http.Request
	LookupPath() string
	Subpath() string
	HasAccessControl() bool
}
