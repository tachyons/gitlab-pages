package serverless

import (
	"errors"
	"net/http/httputil"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// Serverless is a servering used to proxy requests between a client and
// Knative cluster.
type Serverless struct {
	proxy *httputil.ReverseProxy
}

// NewFromAPISource returns a serverless serving instance built from GitLab API
// response
func NewFromAPISource(config api.Serverless) (serving.Serving, error) {
	if len(config.Service) == 0 {
		return nil, errors.New("incomplete serverless serving config")
	}

	certs, err := NewClusterCerts(
		config.Cluster.CertificateCert,
		config.Cluster.CertificateKey,
	)
	if err != nil {
		return nil, err
	}

	cluster := Cluster{
		Name:    config.Cluster.Hostname,
		Address: config.Cluster.Address,
		Port:    config.Cluster.Port,
		Certs:   certs,
	}

	return New(config.Service, cluster), nil
}

// New returns a new serving instance
func New(service string, cluster Cluster) serving.Serving {
	proxy := httputil.ReverseProxy{
		Director:     NewDirectorFunc(service),
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
