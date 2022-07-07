package tls

import (
	"crypto/tls"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/ratelimiter"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

var preferredCipherSuites = []uint16{
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

// GetCertificateFunc returns the certificate to be used for given domain
type GetCertificateFunc func(*tls.ClientHelloInfo) (*tls.Certificate, error)

// GetTLSConfig initializes tls.Config based on config flags
// getCertificateByServerName obtains certificate based on domain
func GetTLSConfig(cfg *config.Config, getCertificateByServerName GetCertificateFunc) (*tls.Config, error) {
	wildcardCertificate, err := tls.X509KeyPair(cfg.General.RootCertificate, cfg.General.RootKey)
	if err != nil {
		return nil, err
	}

	getCertificate := func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		// Golang calls tls.Config.GetCertificate only if it's set and
		// 1. ServerName != ""
		// 2. Or tls.Config.Certificates is empty array
		// tls.Config.Certificates contain wildcard certificate
		// We want to implement rate limits via GetCertificate, so we need to call it every time
		// So we don't set tls.Config.Certificates, but simulate the behavior of golang:
		// 1. try to get certificate by name
		// 2. if we can't, fallback to default(wildcard) certificate
		customCertificate, err := getCertificateByServerName(info)

		if customCertificate != nil || err != nil {
			return customCertificate, err
		}

		return &wildcardCertificate, nil
	}

	TLSDomainRateLimiter := ratelimiter.New(
		"tls_connections_by_domain",
		ratelimiter.WithTLSKeyFunc(ratelimiter.TLSHostnameKey),
		ratelimiter.WithCacheMaxSize(ratelimiter.DefaultDomainCacheSize),
		ratelimiter.WithCachedEntriesMetric(metrics.RateLimitCachedEntries),
		ratelimiter.WithCachedRequestsMetric(metrics.RateLimitCacheRequests),
		ratelimiter.WithBlockedCountMetric(metrics.RateLimitBlockedCount),
		ratelimiter.WithLimitPerSecond(cfg.RateLimit.TLSDomainLimitPerSecond),
		ratelimiter.WithBurstSize(cfg.RateLimit.TLSDomainBurst),
	)

	TLSSourceIPRateLimiter := ratelimiter.New(
		"tls_connections_by_source_ip",
		ratelimiter.WithTLSKeyFunc(ratelimiter.TLSClientIPKey),
		ratelimiter.WithCacheMaxSize(ratelimiter.DefaultSourceIPCacheSize),
		ratelimiter.WithCachedEntriesMetric(metrics.RateLimitCachedEntries),
		ratelimiter.WithCachedRequestsMetric(metrics.RateLimitCacheRequests),
		ratelimiter.WithBlockedCountMetric(metrics.RateLimitBlockedCount),
		ratelimiter.WithLimitPerSecond(cfg.RateLimit.TLSSourceIPLimitPerSecond),
		ratelimiter.WithBurstSize(cfg.RateLimit.TLSSourceIPBurst),
	)

	getCertificate = TLSDomainRateLimiter.GetCertificateMiddleware(getCertificate)
	getCertificate = TLSSourceIPRateLimiter.GetCertificateMiddleware(getCertificate)

	// set MinVersion to fix gosec: G402
	tlsConfig := &tls.Config{GetCertificate: getCertificate, MinVersion: tls.VersionTLS12}

	if !cfg.General.InsecureCiphers {
		tlsConfig.CipherSuites = preferredCipherSuites
	}

	tlsConfig.MinVersion = cfg.TLS.MinVersion
	tlsConfig.MaxVersion = cfg.TLS.MaxVersion

	return tlsConfig, nil
}
