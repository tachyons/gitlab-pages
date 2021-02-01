package httptransport

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
)

// Opt to configure a http.Transport
type Opt func(transport *http.Transport)

type meteredRoundTripper struct {
	next        http.RoundTripper
	name        string
	tracer      *prometheus.HistogramVec
	durations   *prometheus.HistogramVec
	counter     *prometheus.CounterVec
	ttfbTimeout time.Duration
}

// NewMeteredRoundTripper will create a custom http.RoundTripper that can be used with an http.Client.
// The RoundTripper will report metrics based on the collectors passed.
func NewMeteredRoundTripper(name string, tracerVec, durationsVec *prometheus.
	HistogramVec, counterVec *prometheus.CounterVec, ttfbTimeout time.Duration) http.RoundTripper {
	return &meteredRoundTripper{
		next:        InternalTransport,
		name:        name,
		tracer:      tracerVec,
		durations:   durationsVec,
		counter:     counterVec,
		ttfbTimeout: ttfbTimeout,
	}
}

// WithFileProtocol option to be used while ReconfigureMeteredRoundTripper
func WithFileProtocol(protocol string, rt http.RoundTripper) Opt {
	return func(transport *http.Transport) {
		transport.RegisterProtocol(protocol, rt)
	}
}

// ReconfigureMeteredRoundTripper clones meteredRoundTripper and applies options to the transport
func ReconfigureMeteredRoundTripper(rt http.RoundTripper, opts ...Opt) http.RoundTripper {
	mrt, ok := rt.(*meteredRoundTripper)
	if !ok {
		return nil
	}

	t := mrt.next.(*http.Transport)
	for _, opt := range opts {
		opt(t)
	}

	mrt.next = t

	return mrt
}

// withRoundTripper takes an original RoundTripper, reports metrics based on the
// gauge and counter collectors passed
func (mrt *meteredRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	start := time.Now()

	ctx := httptrace.WithClientTrace(r.Context(), mrt.newTracer(start))
	ctx, cancel := context.WithCancel(ctx)

	timer := time.AfterFunc(mrt.ttfbTimeout, cancel)
	defer timer.Stop()

	r = r.WithContext(ctx)

	resp, err := mrt.next.RoundTrip(r)
	if err != nil {
		mrt.counter.WithLabelValues("error").Inc()
		return nil, err
	}

	mrt.logResponse(r, resp)

	statusCode := strconv.Itoa(resp.StatusCode)
	mrt.durations.WithLabelValues(statusCode).Observe(time.Since(start).Seconds())
	mrt.counter.WithLabelValues(statusCode).Inc()

	return resp, nil
}

func (mrt *meteredRoundTripper) logResponse(req *http.Request, resp *http.Response) {
	if log.GetLevel() == log.TraceLevel {
		l := log.WithFields(log.Fields{
			"client_name":     mrt.name,
			"req_url":         req.URL.String(),
			"res_status_code": resp.StatusCode,
		})

		for header, value := range resp.Header {
			l = l.WithField(strings.ToLower(header), strings.Join(value, ";"))
		}

		l.Traceln("response")
	}
}
