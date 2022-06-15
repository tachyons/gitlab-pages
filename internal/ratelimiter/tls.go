package ratelimiter

import (
	"crypto/tls"
	"errors"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/log"
)

var ErrTLSRateLimited = errors.New("too many connections, please retry later")

type GetCertificateFunc func(*tls.ClientHelloInfo) (*tls.Certificate, error)

func (rl *RateLimiter) GetCertificateMiddleware(getCertificate GetCertificateFunc) GetCertificateFunc {
	if rl.limitPerSecond <= 0.0 {
		return getCertificate
	}

	return func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if rl.allowed(rl.tlsKeyFunc(info)) {
			return getCertificate(info)
		}

		rl.logRateLimitedTLS(info)

		if rl.blockedCount != nil {
			rl.blockedCount.WithLabelValues(rl.name).Inc()
		}

		return nil, ErrTLSRateLimited
	}
}

func (rl *RateLimiter) logRateLimitedTLS(info *tls.ClientHelloInfo) {
	log.WithFields(logrus.Fields{
		"rate_limiter_name":             rl.name,
		"source_ip":                     TLSClientIPKey(info),
		"req_host":                      info.ServerName,
		"rate_limiter_limit_per_second": rl.limitPerSecond,
		"rate_limiter_burst_size":       rl.burstSize,
	}).Info("TLS connection rate-limited")
}
