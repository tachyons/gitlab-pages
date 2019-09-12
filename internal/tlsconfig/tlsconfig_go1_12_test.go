// +build go1.12

package tlsconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnableTLS13(t *testing.T) {
	tests := map[string]struct {
		tlsMin      string
		tlsMax      string
		enableTLS13 bool
	}{
		"ask for minimum TLS 1.3": {tlsMin: "tls1.3", tlsMax: "", enableTLS13: true},
		"ask for maximim TLS 1.3": {tlsMin: "", tlsMax: "tls1.3", enableTLS13: true},
		"do not ask for TLS 1.3":  {tlsMin: "tls1.2", tlsMax: "tls1.2", enableTLS13: false},
	}

	// Store original GODEBUG value
	godebug := os.Getenv("GODEBUG")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateTLSVersions(tc.tlsMin, tc.tlsMax)
			require.NoError(t, err)

			if tc.enableTLS13 {
				require.Regexp(t, "tls13=1", os.Getenv("GODEBUG"))
			} else {
				require.NotRegexp(t, "tls13=1", godebug)
			}
		})

		// Restore original GODEBUG value
		os.Setenv("GODEBUG", godebug)
	}
}
