package acceptance_test

import (
	"crypto/tls"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcceptsSupportedCiphers(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpsListener}),
	)

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
	require.NoError(t, err)

	t.Cleanup(func() {
		rsp.Body.Close()
	})
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
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpsListener}),
	)

	client, cleanup := ClientWithConfig(tlsConfigWithInsecureCiphersOnly())
	defer cleanup()

	rsp, err := client.Get(httpsListener.URL("/"))
	require.Nil(t, rsp)
	require.Error(t, err)
}

func TestEnableInsecureCiphers(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpsListener}),
		withExtraArgument("-insecure-ciphers", "true"),
	)

	client, cleanup := ClientWithConfig(tlsConfigWithInsecureCiphersOnly())
	defer cleanup()

	rsp, err := client.Get(httpsListener.URL("/"))
	require.NoError(t, err)
	t.Cleanup(func() {
		rsp.Body.Close()
	})
}

func TestTLSVersions(t *testing.T) {
	tests := map[string]struct {
		tlsMin      string
		tlsMax      string
		tlsClient   uint16
		expectError bool
	}{
		"client version not supported":             {tlsMin: "tls1.2", tlsMax: "tls1.3", tlsClient: tls.VersionTLS10, expectError: true},
		"client version supported":                 {tlsMin: "tls1.2", tlsMax: "tls1.3", tlsClient: tls.VersionTLS12, expectError: false},
		"client and server using default settings": {tlsMin: "", tlsMax: "", tlsClient: 0, expectError: false},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			var args []string
			if tc.tlsMin != "" {
				args = append(args, "-tls-min-version", tc.tlsMin)
			}
			if tc.tlsMax != "" {
				args = append(args, "-tls-max-version", tc.tlsMax)
			}

			RunPagesProcess(t,
				withListeners([]ListenSpec{httpsListener}),
				withArguments(args),
			)

			tlsConfig := &tls.Config{}
			if tc.tlsClient != 0 {
				tlsConfig.MinVersion = tc.tlsClient
				tlsConfig.MaxVersion = tc.tlsClient
			}
			client, cleanup := ClientWithConfig(tlsConfig)
			defer cleanup()

			rsp, err := client.Get(httpsListener.URL("/"))

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				rsp.Body.Close()
			}
		})
	}
}
