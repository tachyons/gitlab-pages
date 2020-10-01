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
			httpTraceObserve("httptrace.ClientTrace.GetConn", start)

			log.WithFields(log.Fields{
				"host": host,
			}).Traceln("httptrace.ClientTrace.GetConn")
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			httpTraceObserve("httptrace.ClientTrace.GotConn", start)

			log.WithFields(log.Fields{
				"reused":       connInfo.Reused,
				"was_idle":     connInfo.WasIdle,
				"idle_time_ms": connInfo.IdleTime.Milliseconds(),
			}).Traceln("httptrace.ClientTrace.GotConn")
		},
		GotFirstResponseByte: func() {
			httpTraceObserve("httptrace.ClientTrace.GotFirstResponseByte", start)
		},
		DNSStart: func(d httptrace.DNSStartInfo) {
			httpTraceObserve("httptrace.ClientTrace.DNSStart", start)
		},
		DNSDone: func(d httptrace.DNSDoneInfo) {
			httpTraceObserve("httptrace.ClientTrace.DNSDone", start)

			log.WithFields(log.Fields{}).WithError(d.Err).
				Traceln("httptrace.ClientTrace.DNSDone")
		},
		ConnectStart: func(net, addr string) {
			httpTraceObserve("httptrace.ClientTrace.ConnectStart", start)

			log.WithFields(log.Fields{
				"network": net,
				"address": addr,
			}).Traceln("httptrace.ClientTrace.ConnectStart")
		},
		ConnectDone: func(net string, addr string, err error) {
			httpTraceObserve("httptrace.ClientTrace.ConnectDone", start)

			log.WithFields(log.Fields{
				"network": net,
				"address": addr,
			}).WithError(err).Traceln("httptrace.ClientTrace.ConnectDone")
		},
		TLSHandshakeStart: func() {
			httpTraceObserve("httptrace.ClientTrace.TLSHandshakeStart", start)
		},
		TLSHandshakeDone: func(connState tls.ConnectionState, err error) {
			httpTraceObserve("httptrace.ClientTrace.TLSHandshakeDone", start)

			log.WithFields(log.Fields{
				"version":            connState.Version,
				"connection_resumed": connState.DidResume,
			}).WithError(err).Traceln("httptrace.ClientTrace.TLSHandshakeDone")
		},
	}

	return trace
}

func httpTraceObserve(label string, start time.Time) {
	metrics.ObjectStorageTraceDuration.WithLabelValues(label).
		Observe(time.Since(start).Seconds())
}
