package acceptance_test

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcceptsSupportedCiphers(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, supportedListeners(), "")
	defer teardown()

	tlsConfig := &tls.Config{
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		},
	}
	client, cleanup := ClientWithConfig(tlsConfig)
	defer cleanup()

	rsp, err := client.Get(httpsListener.URL("/"))

	if rsp != nil {
		rsp.Body.Close()
	}

	require.NoError(t, err)
}

func tlsConfigWithInsecureCiphersOnly() *tls.Config {
	return &tls.Config{
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		},
		MaxVersion: tls.VersionTLS12, // ciphers for TLS1.3 are not configurable and will work if enabled
	}
}

func TestRejectsUnsupportedCiphers(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, supportedListeners(), "")
	defer teardown()

	client, cleanup := ClientWithConfig(tlsConfigWithInsecureCiphersOnly())
	defer cleanup()

	rsp, err := client.Get(httpsListener.URL("/"))

	if rsp != nil {
		rsp.Body.Close()
	}

	require.Error(t, err)
	require.Nil(t, rsp)
}

func TestEnableInsecureCiphers(t *testing.T) {
	skipUnlessEnabled(t)
	teardown := RunPagesProcess(t, *pagesBinary, supportedListeners(), "", "-insecure-ciphers")
	defer teardown()

	client, cleanup := ClientWithConfig(tlsConfigWithInsecureCiphersOnly())
	defer cleanup()

	rsp, err := client.Get(httpsListener.URL("/"))

	if rsp != nil {
		rsp.Body.Close()
	}

	require.NoError(t, err)
}

func TestTLSVersions(t *testing.T) {
	skipUnlessEnabled(t)

	tests := map[string]struct {
		tlsMin      string
		tlsMax      string
		tlsClient   uint16
		expectError bool
	}{
		"client version not supported":             {tlsMin: "tls1.1", tlsMax: "tls1.2", tlsClient: tls.VersionTLS10, expectError: true},
		"client version supported":                 {tlsMin: "tls1.1", tlsMax: "tls1.2", tlsClient: tls.VersionTLS12, expectError: false},
		"client and server using default settings": {tlsMin: "", tlsMax: "", tlsClient: 0, expectError: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			args := []string{}
			if tc.tlsMin != "" {
				args = append(args, "-tls-min-version", tc.tlsMin)
			}
			if tc.tlsMax != "" {
				args = append(args, "-tls-max-version", tc.tlsMax)
			}

			teardown := RunPagesProcess(t, *pagesBinary, supportedListeners(), "", args...)
			defer teardown()

			tlsConfig := &tls.Config{}
			if tc.tlsClient != 0 {
				tlsConfig.MinVersion = tc.tlsClient
				tlsConfig.MaxVersion = tc.tlsClient
			}
			client, cleanup := ClientWithConfig(tlsConfig)
			defer cleanup()

			rsp, err := client.Get(httpsListener.URL("/"))

			if rsp != nil {
				rsp.Body.Close()
			}

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
