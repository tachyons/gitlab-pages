package customheaders

import (
	"net/http"
)

// AddCustomHeaders adds a map of Headers to a Response
func AddCustomHeaders(w http.ResponseWriter, headers http.Header) {
	for k, v := range headers {
		for _, value := range v {
			w.Header().Add(k, value)
		}
	}
}
