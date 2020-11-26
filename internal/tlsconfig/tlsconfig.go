package tlsconfig

import (
	"crypto/tls"
	"fmt"
	"sort"
	"strings"
)

// GetCertificateFunc returns the certificate to be used for given domain
type GetCertificateFunc func(*tls.ClientHelloInfo) (*tls.Certificate, error)

var (
	preferredCipherSuites = []uint16{
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_AES_128_GCM_SHA256,
		tls.TLS_AES_256_GCM_SHA384,
		tls.TLS_CHACHA20_POLY1305_SHA256,
	}

	// AllTLSVersions has all supported flag values
	AllTLSVersions = map[string]uint16{
		"":       0, // Default value in tls.Config
		"ssl3":   tls.VersionSSL30,
		"tls1.0": tls.VersionTLS10,
		"tls1.1": tls.VersionTLS11,
		"tls1.2": tls.VersionTLS12,
		"tls1.3": tls.VersionTLS13,
	}
)

// FlagUsage returns string with explanation how to use the CLI flag
func FlagUsage(minOrMax string) string {
	versions := []string{}

	for version := range AllTLSVersions {
		if version != "" {
			versions = append(versions, fmt.Sprintf("%q", version))
		}
	}
	sort.Strings(versions)

	return fmt.Sprintf("Specifies the "+minOrMax+"imum SSL/TLS version, supported values are %s", strings.Join(versions, ", "))
}

// Create returns tls.Config for given app configuration
func Create(cert, key []byte, getCertificate GetCertificateFunc, insecureCiphers bool, tlsMinVersion uint16, tlsMaxVersion uint16) (*tls.Config, error) {
	tlsConfig := &tls.Config{GetCertificate: getCertificate}

	err := configureCertificate(tlsConfig, cert, key)
	if err != nil {
		return nil, err
	}

	if !insecureCiphers {
		configureTLSCiphers(tlsConfig)
	}

	tlsConfig.MinVersion = tlsMinVersion
	tlsConfig.MaxVersion = tlsMaxVersion

	return tlsConfig, nil
}

// ValidateTLSVersions returns error if the provided TLS versions config values are not valid
func ValidateTLSVersions(min, max string) error {
	tlsMin, tlsMinOk := AllTLSVersions[min]
	tlsMax, tlsMaxOk := AllTLSVersions[max]

	if !tlsMinOk {
		return fmt.Errorf("invalid minimum TLS version: %s", min)
	}
	if !tlsMaxOk {
		return fmt.Errorf("invalid maximum TLS version: %s", max)
	}
	if tlsMin > tlsMax && tlsMax > 0 {
		return fmt.Errorf("invalid maximum TLS version: %s; should be at least %s", max, min)
	}

	return nil
}

func configureCertificate(tlsConfig *tls.Config, cert, key []byte) error {
	certificate, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return err
	}

	tlsConfig.Certificates = []tls.Certificate{certificate}

	return nil
}

func configureTLSCiphers(tlsConfig *tls.Config) {
	tlsConfig.PreferServerCipherSuites = true
	tlsConfig.CipherSuites = preferredCipherSuites
}
