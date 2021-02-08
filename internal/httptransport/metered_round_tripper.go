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

// MeteredRoundTripper is a custom http.Transport that implements the http.RoundTripper interface.
// It holds prometheus metrics to report connection usage and durations.
type MeteredRoundTripper struct {
	next        http.RoundTripper
	name        string
	tracer      *prometheus.HistogramVec
	durations   *prometheus.HistogramVec
	counter     *prometheus.CounterVec
	ttfbTimeout time.Duration
}

// NewMeteredRoundTripper will create a custom http.RoundTripper that can be used with an http.Client.
// The RoundTripper will report metrics based on the collectors passed.
func NewMeteredRoundTripper(transport *http.Transport, name string, tracerVec, durationsVec *prometheus.
	HistogramVec, counterVec *prometheus.CounterVec, ttfbTimeout time.Duration) http.RoundTripper {
	if transport == nil {
		transport = DefaultTransport
	}

	return &MeteredRoundTripper{
		next:        transport,
		name:        name,
		tracer:      tracerVec,
		durations:   durationsVec,
		counter:     counterVec,
		ttfbTimeout: ttfbTimeout,
	}
}

// RoundTrip wraps the original http.Transport into a MeteredRoundTripper which
// reports metrics on request duration, tracing and request count
func (mrt *MeteredRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
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

func (mrt *MeteredRoundTripper) logResponse(req *http.Request, resp *http.Response) {
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

// RegisterProtocol calls the RegisterProtocol on the MeteredRoundTripper's next Transport
func (mrt *MeteredRoundTripper) RegisterProtocol(scheme string, rt http.RoundTripper) {
	mrt.next.(*http.Transport).RegisterProtocol(scheme, rt)
}
