package httptransport

import (
	"context"
	"net/http"
	"net/http/httptrace"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
)

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
func NewMeteredRoundTripper(transport http.RoundTripper, name string, tracerVec, durationsVec *prometheus.
	HistogramVec, counterVec *prometheus.CounterVec, ttfbTimeout time.Duration) http.RoundTripper {
	if transport == nil {
		transport = DefaultTransport
	}

	return &meteredRoundTripper{
		next:        transport,
		name:        name,
		tracer:      tracerVec,
		durations:   durationsVec,
		counter:     counterVec,
		ttfbTimeout: ttfbTimeout,
	}
}

// RoundTrip wraps the original http.Transport into a meteredRoundTripper which
// reports metrics on request duration, tracing and request count
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
	if logrus.GetLevel() == logrus.TraceLevel {
		logging.LogRequest(req).WithFields(logrus.Fields{
			"client_name":     mrt.name,
			"req_url":         req.URL.String(),
			"res_status_code": resp.StatusCode,
			"res_headers":     resp.Header,
		}).Traceln("response from client")
	}
}

// RegisterProtocol allows to call RegisterProtocol on the meteredRoundTripper's transport
// outside of this package
func (mrt *meteredRoundTripper) RegisterProtocol(scheme string, rt http.RoundTripper) {
	mrt.next.(*http.Transport).RegisterProtocol(scheme, rt)
}
