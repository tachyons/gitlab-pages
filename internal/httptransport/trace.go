package httptransport

import (
	"crypto/tls"
	"net/http/httptrace"
	"time"

	"gitlab.com/gitlab-org/labkit/log"
)

func (mrt *meteredRoundTripper) newTracer(start time.Time) *httptrace.
	ClientTrace {
	trace := &httptrace.ClientTrace{
		GetConn: func(host string) {
			mrt.httpTraceObserve("httptrace.ClientTrace.GetConn", start)

			log.WithFields(log.Fields{
				"host": host,
			}).Traceln("httptrace.ClientTrace.GetConn")
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			mrt.httpTraceObserve("httptrace.ClientTrace.GotConn", start)

			log.WithFields(log.Fields{
				"reused":       connInfo.Reused,
				"was_idle":     connInfo.WasIdle,
				"idle_time_ms": connInfo.IdleTime.Milliseconds(),
			}).Traceln("httptrace.ClientTrace.GotConn")
		},
		GotFirstResponseByte: func() {
			mrt.httpTraceObserve("httptrace.ClientTrace.GotFirstResponseByte", start)
		},
		DNSStart: func(d httptrace.DNSStartInfo) {
			mrt.httpTraceObserve("httptrace.ClientTrace.DNSStart", start)
		},
		DNSDone: func(d httptrace.DNSDoneInfo) {
			mrt.httpTraceObserve("httptrace.ClientTrace.DNSDone", start)

			log.WithFields(log.Fields{}).WithError(d.Err).
				Traceln("httptrace.ClientTrace.DNSDone")
		},
		ConnectStart: func(net, addr string) {
			mrt.httpTraceObserve("httptrace.ClientTrace.ConnectStart", start)

			log.WithFields(log.Fields{
				"network": net,
				"address": addr,
			}).Traceln("httptrace.ClientTrace.ConnectStart")
		},
		ConnectDone: func(net string, addr string, err error) {
			mrt.httpTraceObserve("httptrace.ClientTrace.ConnectDone", start)

			l := log.WithFields(log.Fields{
				"network": net,
				"address": addr,
			})

			if err != nil {
				l.WithError(err).Error("httptrace.ClientTrace.ConnectDone")
			}

			l.Traceln("httptrace.ClientTrace.ConnectDone")
		},
		TLSHandshakeStart: func() {
			mrt.httpTraceObserve("httptrace.ClientTrace.TLSHandshakeStart", start)
		},
		TLSHandshakeDone: func(connState tls.ConnectionState, err error) {
			mrt.httpTraceObserve("httptrace.ClientTrace.TLSHandshakeDone", start)

			l := log.WithFields(log.Fields{
				"version":            connState.Version,
				"connection_resumed": connState.DidResume,
			})

			if err != nil {
				l.WithError(err).Error("httptrace.ClientTrace.TLSHandshakeDone")
			}

			l.Traceln("httptrace.ClientTrace.TLSHandshakeDone")
		},
	}

	return trace
}

func (mrt *meteredRoundTripper) httpTraceObserve(label string, start time.Time) {
	mrt.tracer.WithLabelValues(label).
		Observe(time.Since(start).Seconds())
}
