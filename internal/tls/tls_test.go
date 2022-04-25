package tls

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
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

func TestInvalidKeyPair(t *testing.T) {
	cfg := &config.Config{}
	_, err := GetTLSConfig(cfg, getCertificate)
	require.EqualError(t, err, "tls: failed to find any PEM data in certificate input")
}

func TestInsecureCiphers(t *testing.T) {
	cfg := &config.Config{
		General: config.General{
			RootCertificate: cert,
			RootKey:         key,
			InsecureCiphers: true,
		},
	}
	tlsConfig, err := GetTLSConfig(cfg, getCertificate)
	require.NoError(t, err)
	require.Empty(t, tlsConfig.CipherSuites)
}

func TestGetTLSConfig(t *testing.T) {
	cfg := &config.Config{
		General: config.General{
			RootCertificate: cert,
			RootKey:         key,
		},
		TLS: config.TLS{
			MinVersion: tls.VersionTLS11,
			MaxVersion: tls.VersionTLS12,
		},
	}
	tlsConfig, err := GetTLSConfig(cfg, getCertificate)
	require.NoError(t, err)
	require.IsType(t, getCertificate, tlsConfig.GetCertificate)
	require.Equal(t, preferredCipherSuites, tlsConfig.CipherSuites)
	require.Equal(t, uint16(tls.VersionTLS11), tlsConfig.MinVersion)
	require.Equal(t, uint16(tls.VersionTLS12), tlsConfig.MaxVersion)

	cert, err := tlsConfig.GetCertificate(&tls.ClientHelloInfo{})
	require.NoError(t, err)
	require.NotNil(t, cert)
}
