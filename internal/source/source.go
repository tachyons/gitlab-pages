package source

import "net/http"

// Source interface is a common interface between all sources in the source
// package.
type Source interface { // TODO define me better
	Serve(w http.ResponseWriter, r *http.Request) bool
}
