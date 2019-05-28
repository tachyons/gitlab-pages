// +build go1.12

package tlsconfig

import (
	"crypto/tls"
)

func init() {
	AllTLSVersions["tls1.3"] = tls.VersionTLS13

	preferredCipherSuites = append(preferredCipherSuites,
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	)
}
