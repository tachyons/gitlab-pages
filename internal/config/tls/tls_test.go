package tls

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

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
		"TLS versions conflict":       {tlsMin: "tls1.3", tlsMax: "tls1.2", err: "invalid maximum TLS version: tls1.2; should be at least tls1.3"},
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

func TestVerifyCert(t *testing.T) {
	tests := map[string]struct {
		domainName     string
		certBytes      []byte
		expectedErrMsg string
	}{
		"empty_cert_bytes_no_error": {domainName: "gitlab.io", certBytes: nil, expectedErrMsg: ""},
		"invalid_cert_bytes":        {domainName: "gitlab.io", certBytes: []byte(`not PEM bytes`), expectedErrMsg: ErrEmptyCert.Error()},
		"valid_cert":                {domainName: "gitlab.io", certBytes: genTestCert(t, "gitlab.io", time.Second)},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := VerifyCert(test.domainName, test.certBytes)
			if test.expectedErrMsg != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErrMsg)
				return
			}

			require.NoError(t, err)
		})
	}
}

func genTestCert(t *testing.T, commonName string, expiry time.Duration) []byte {
	t.Helper()

	priv, err := ecdsa.GenerateKey(elliptic.P521(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		DNSNames:     []string{commonName},
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization: []string{"Acme Co"},
			CommonName:   commonName,
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(expiry),

		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		KeyUs
	}

	/*
	   hosts := strings.Split(*host, ",")
	   for _, h := range hosts {
	   	if ip := net.ParseIP(h); ip != nil {
	   		template.IPAddresses = append(template.IPAddresses, ip)
	   	} else {
	   		template.DNSNames = append(template.DNSNames, h)
	   	}
	   }
	   if *isCA {
	   	template.IsCA = true
	   	template.KeyUsage |= x509.KeyUsageCertSign
	   }
	*/

	derBytes, err := x509.CreateCertificate(rand.Reader, &template, &template, publicKey(priv), priv)
	require.NoError(t, err)
	//return derBytes
	//if err != nil {
	//	log.Fatalf("Failed to create certificate: %s", err)
	//}
	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: "CERTIFICATE", Bytes: derBytes})
	////fmt.Println(out.String())
	//out.Reset()
	pem.Encode(out, pemBlockForKey(priv))
	//fmt.Println(out.String())
	return out.Bytes()
}

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: "EC PRIVATE KEY", Bytes: b}
	default:
		return nil
	}
}
