package ratelimiter

import (
	"crypto/tls"
	"errors"
	"net"
	"strconv"

	"github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/log"

	tlsconfig "gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
)

var TLSRateLimitedError = errors.New("TLS connection is being rate-limited")

func (rl *RateLimiter) GetCertificateMiddleware(getCertificate tlsconfig.GetCertificateFunc) tlsconfig.GetCertificateFunc {
	log.Info("GetCertificateMiddleware set")
	return func(hi *tls.ClientHelloInfo) (*tls.Certificate, error) {
		log.WithFields(logrus.Fields{
			"server_name": hi.ServerName,
		}).Info("GetCertificateMiddleware called")

		return getCertificate(hi)

		if rl.allowed(hi.ServerName) {
			return getCertificate(hi)
		}

		rl.logRateLimitedTLS(hi)

		if rl.blockedCount != nil {
			rl.blockedCount.WithLabelValues(strconv.FormatBool(rl.enforce)).Inc()
		}

		if !rl.enforce {
			return getCertificate(hi)
		}

		return nil, TLSRateLimitedError
	}
}

func (rl *RateLimiter) logRateLimitedTLS(hi *tls.ClientHelloInfo) {
	log.WithFields(logrus.Fields{
		"rate_limiter_name":             rl.name,
		"source_ip":                     getRemoteAddrFromHelloInfo(hi),
		"req_host":                      hi.ServerName,
		"rate_limiter_limit_per_second": rl.limitPerSecond,
		"rate_limiter_burst_size":       rl.burstSize,
	}).Info("TLS connection rate-limited")
}

func getRemoteAddrFromHelloInfo(hi *tls.ClientHelloInfo) string {
	remoteAddr := hi.Conn.RemoteAddr().String()
	remoteAddr, _, err := net.SplitHostPort(remoteAddr)
	if err != nil {
		return remoteAddr
	}

	return remoteAddr
}
