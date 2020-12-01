package tlsconfig

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

var cert = []byte(`-----BEGIN CERTIFICATE-----
MIIBhTCCASugAwIBAgIQIRi6zePL6mKjOipn+dNuaTAKBggqhkjOPQQDAjASMRAw
DgYDVQQKEwdBY21lIENvMB4XDTE3MTAyMDE5NDMwNloXDTE4MTAyMDE5NDMwNlow
EjEQMA4GA1UEChMHQWNtZSBDbzBZMBMGByqGSM49AgEGCCqGSM49AwEHA0IABD0d
7VNhbWvZLWPuj/RtHFjvtJBEwOkhbN/BnnE8rnZR8+sbwnc/KhCk3FhnpHZnQz7B
5aETbbIgmuvewdjvSBSjYzBhMA4GA1UdDwEB/wQEAwICpDATBgNVHSUEDDAKBggr
BgEFBQcDATAPBgNVHRMBAf8EBTADAQH/MCkGA1UdEQQiMCCCDmxvY2FsaG9zdDo1
NDUzgg4xMjcuMC4wLjE6NTQ1MzAKBggqhkjOPQQDAgNIADBFAiEA2zpJEPQyz6/l
Wf86aX6PepsntZv2GYlA5UpabfT2EZICICpJ5h/iI+i341gBmLiAFQOyTDT+/wQc
6MF9+Yw1Yy0t
-----END CERTIFICATE-----`)

var key = []byte(`-----BEGIN EC PRIVATE KEY-----
MHcCAQEEIIrYSSNQFaA2Hwf1duRSxKtLYX5CB04fSeQ6tF1aY/PuoAoGCCqGSM49
AwEHoUQDQgAEPR3tU2Fta9ktY+6P9G0cWO+0kETA6SFs38GecTyudlHz6xvCdz8q
EKTcWGekdmdDPsHloRNtsiCa697B2O9IFA==
-----END EC PRIVATE KEY-----`)

var getCertificate = func(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return nil, nil
}

func TestValidateTLSVersions(t *testing.T) {
	tests := map[string]struct {
		tlsMin string
		tlsMax string
		err    string
	}{
		"invalid minimum TLS version": {tlsMin: "tls123", tlsMax: "", err: "invalid minimum TLS version: tls123"},
		"invalid maximum TLS version": {tlsMin: "", tlsMax: "tls123", err: "invalid maximum TLS version: tls123"},
		"TLS versions conflict":       {tlsMin: "tls1.2", tlsMax: "tls1.1", err: "invalid maximum TLS version: tls1.1; should be at least tls1.2"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			err := ValidateTLSVersions(tc.tlsMin, tc.tlsMax)
			require.EqualError(t, err, tc.err)
		})
	}
}

func TestInvalidKeyPair(t *testing.T) {
	_, err := Create([]byte(``), []byte(``), getCertificate, false, tls.VersionTLS11, tls.VersionTLS12)
	require.EqualError(t, err, "tls: failed to find any PEM data in certificate input")
}

func TestInsecureCiphers(t *testing.T) {
	tlsConfig, err := Create(cert, key, getCertificate, true, tls.VersionTLS11, tls.VersionTLS12)
	require.NoError(t, err)
	require.False(t, tlsConfig.PreferServerCipherSuites)
	require.Empty(t, tlsConfig.CipherSuites)
}

func TestCreate(t *testing.T) {
	tlsConfig, err := Create(cert, key, getCertificate, false, tls.VersionTLS11, tls.VersionTLS12)
	require.NoError(t, err)
	require.IsType(t, getCertificate, tlsConfig.GetCertificate)
	require.True(t, tlsConfig.PreferServerCipherSuites)
	require.Equal(t, preferredCipherSuites, tlsConfig.CipherSuites)
	require.Equal(t, uint16(tls.VersionTLS11), tlsConfig.MinVersion)
	require.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MaxVersion)
}
