package ratelimiter

import (
	"crypto/tls"
	"errors"
	"strconv"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/log"

	tlsconfig "gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
)

var ErrTLSRateLimited = errors.New("too many connections, please retry later")

func (rl *RateLimiter) GetCertificateMiddleware(getCertificate tlsconfig.GetCertificateFunc) tlsconfig.GetCertificateFunc {
	if rl.limitPerSecond <= 0.0 {
		return getCertificate
	}

	return func(info *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if rl.allowed(rl.tlsKeyFunc(info)) {
			return getCertificate(info)
		}

		rl.logRateLimitedTLS(info)

		if rl.blockedCount != nil {
			rl.blockedCount.WithLabelValues(rl.name, strconv.FormatBool(rl.enforce)).Inc()
		}

		if !rl.enforce {
			return getCertificate(info)
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
		"enforced":                      rl.enforce,
	}).Info("TLS connection rate-limited")
}
