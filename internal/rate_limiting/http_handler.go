package rate_limiting

import (
	"crypto/tls"
	"errors"
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal/tlsconfig"
)

func (r *RateLimiting) LimitHostHandler(handler http.Handler) http.Handler {
	fn := func(rw http.ResponseWriter, req *http.Request) {
		if r.Allow(req.Host) {
			handler.ServeHTTP(rw, req)
			return
		}

		rw.WriteHeader(http.StatusTooManyRequests)
	}

	return http.HandlerFunc(fn)
}

func (r *RateLimiting) LimitServeTLS(handler tlsconfig.GetCertificateFunc) tlsconfig.GetCertificateFunc {
	return func(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
		if r.Allow(ch.ServerName) {
			return handler(ch)
		}

		return nil, errors.New("rate limited")
	}
}
