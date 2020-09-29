package httprange

import (
	"crypto/tls"
	"net/http"
	"net/http/httptrace"
	"time"

	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

type tracedTransport struct {
	next http.RoundTripper
}

// withRoundTripper takes an original RoundTripper, reports metrics based on the
// gauge and counter collectors passed
func (tr *tracedTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.WithContext(httptrace.WithClientTrace(r.Context(), newTracer(time.Now())))

	return tr.next.RoundTrip(r)
}

func newTracer(start time.Time) *httptrace.ClientTrace {
	trace := &httptrace.ClientTrace{
		GetConn: func(host string) {
			metrics.ObjectStorageResponsiveness.WithLabelValues("get_connection").Observe(float64(time.Since(start)))

			log.WithFields(log.Fields{
				"host": host,
			}).Traceln("get_connection")
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			metrics.ObjectStorageResponsiveness.WithLabelValues("get_connection").Observe(float64(time.Since(start)))

			log.WithFields(log.Fields{
				"reused":       connInfo.Reused,
				"was_idle":     connInfo.WasIdle,
				"idle_time_ms": connInfo.IdleTime.Milliseconds(),
			}).Traceln("got_connection")
		},
		PutIdleConn: nil,
		GotFirstResponseByte: func() {
			metrics.ObjectStorageResponsiveness.WithLabelValues("got_first_response_byte").Observe(float64(time.Since(start)))
		},
		Got100Continue: nil,
		Got1xxResponse: nil,
		DNSStart: func(d httptrace.DNSStartInfo) {
			metrics.ObjectStorageResponsiveness.WithLabelValues("dns_lookup_start").Observe(float64(time.Since(start)))
		},
		DNSDone: func(d httptrace.DNSDoneInfo) {
			metrics.ObjectStorageResponsiveness.WithLabelValues("dns_lookup_done").Observe(float64(time.Since(start)))

			log.WithFields(log.Fields{}).WithError(d.Err).Traceln("connect_start")
		},
		ConnectStart: func(net, addr string) {
			metrics.ObjectStorageResponsiveness.WithLabelValues("connect_start").Observe(float64(time.Since(start)))

			log.WithFields(log.Fields{
				"network": net,
				"address": addr,
			}).Traceln("connect_start")
		},
		ConnectDone: func(net string, addr string, err error) {
			metrics.ObjectStorageResponsiveness.WithLabelValues("connect_done").Observe(float64(time.Since(start)))

			log.WithFields(log.Fields{
				"network": net,
				"address": addr,
			}).WithError(err).Traceln("connect_done")
		},
		TLSHandshakeStart: func() {
			metrics.ObjectStorageResponsiveness.WithLabelValues("tls_handshake_start").Observe(float64(time.Since(start)))
		},
		TLSHandshakeDone: func(connState tls.ConnectionState, err error) {
			metrics.ObjectStorageResponsiveness.WithLabelValues("tls_handshake_done").Observe(float64(time.Since(start)))

			log.WithFields(log.Fields{
				"version":            connState.Version,
				"connection_resumed": connState.DidResume,
			}).WithError(err).Traceln("tls_handshake_done")
		},
		WroteHeaderField: nil,
		WroteHeaders:     nil,
		Wait100Continue:  nil,
		WroteRequest:     nil,
	}

	return trace
}
