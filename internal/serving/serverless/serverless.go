package serverless

import (
	"net/http/httputil"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Serverless is a servering used to proxy requests between a client and
// Knative cluster.
type Serverless struct {
	proxy *httputil.ReverseProxy
}

// New returns a new serving instance
func New(function Function, cluster Cluster) serving.Serving {
	proxy := httputil.ReverseProxy{
		Director:     NewDirectorFunc(function),
		Transport:    NewTransport(cluster),
		ErrorHandler: NewErrorHandler(),
	}

	return &Serverless{proxy: &proxy}
}

// ServeFileHTTP handle an incoming request and proxies it to Knative cluster
func (s *Serverless) ServeFileHTTP(h serving.Handler) bool {
	metrics.ServerlessRequests.Inc()

	s.proxy.ServeHTTP(h.Writer, h.Request)

	return true
}

// ServeNotFoundHTTP responds with 404
func (s *Serverless) ServeNotFoundHTTP(h serving.Handler) {
	httperrors.Serve404(h.Writer)
}
