// +build !go1.14

package httptransport

import (
    "crypto/tls"
    "net/http"
)

func clone(t *http.Transport) *http.Transport {
	// t.Clone() panics in Go 1.13 due to tls.Config being empty
	// so we need to explicitly initialize it
	// https://github.com/golang/go/issues/40565
	if t.TLSClientConfig == nil {
		t.TLSClientConfig = &tls.Config{}
	}

	return t.Clone()
}
