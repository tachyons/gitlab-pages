// +build go1.13,!go1.14

package tlsconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDisableTLS13(t *testing.T) {
	tests := map[string]struct {
		tlsMin       string
		tlsMax       string
		disableTLS13 bool
	}{
		"ask for minimum TLS 1.3": {tlsMin: "tls1.3", tlsMax: "", disableTLS13: true},
		"ask for maximum TLS 1.3": {tlsMin: "", tlsMax: "tls1.3", disableTLS13: true},
		"do not ask for TLS 1.3":  {tlsMin: "tls1.2", tlsMax: "tls1.2", disableTLS13: false},
	}

	// Store original GODEBUG value
	godebug := os.Getenv("GODEBUG")
	defer os.Setenv("GODEBUG", godebug)

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			if tc.disableTLS13 {
				os.Setenv("GODEBUG", "tls13=0")
			}
			err := ValidateTLSVersions(tc.tlsMin, tc.tlsMax)
			if tc.disableTLS13 {
				require.EqualError(t, err, "tls1.3 is disabled: GODEBUG=tls13=0")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
