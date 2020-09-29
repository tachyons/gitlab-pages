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
			observe("get_connection", start)

			log.WithFields(log.Fields{
				"host": host,
			}).Traceln("get_connection")
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			observe("get_connection", start)

			log.WithFields(log.Fields{
				"reused":       connInfo.Reused,
				"was_idle":     connInfo.WasIdle,
				"idle_time_ms": connInfo.IdleTime.Milliseconds(),
			}).Traceln("get_connection")
		},
		PutIdleConn: nil,
		GotFirstResponseByte: func() {
			observe("got_first_response_byte", start)
		},
		Got100Continue: nil,
		Got1xxResponse: nil,
		DNSStart: func(d httptrace.DNSStartInfo) {
			observe("dns_lookup_start", start)
		},
		DNSDone: func(d httptrace.DNSDoneInfo) {
			observe("dns_lookup_done", start)

			log.WithFields(log.Fields{}).WithError(d.Err).Traceln("dns_lookup_done")
		},
		ConnectStart: func(net, addr string) {
			observe("connect_start", start)

			log.WithFields(log.Fields{
				"network": net,
				"address": addr,
			}).Traceln("connect_start")
		},
		ConnectDone: func(net string, addr string, err error) {
			observe("connect_done", start)

			log.WithFields(log.Fields{
				"network": net,
				"address": addr,
			}).WithError(err).Traceln("connect_done")
		},
		TLSHandshakeStart: func() {
			observe("tls_handshake_start", start)
		},
		TLSHandshakeDone: func(connState tls.ConnectionState, err error) {
			observe("tls_handshake_done", start)

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

func observe(label string, start time.Time) {
	metrics.ObjectStorageResponsiveness.WithLabelValues(label).Observe(float64(time.Since(start).Milliseconds()))
}
