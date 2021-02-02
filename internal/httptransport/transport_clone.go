// +build go1.14

package httptransport

import (
	"net/http"
)

// from go 1.14 onwards call Clone directly
func clone(t *http.Transport) *http.Transport {
	return t.Clone()
}
