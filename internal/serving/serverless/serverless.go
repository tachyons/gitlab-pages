package serverless

import (
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving"
)

// Serverless is a servering used to proxy requests between a client and
// Knative cluster.
type Serverless struct {
}

// ServeFileHTTP handle an incoming request and proxies it to Knative cluster
func (s *Serverless) ServeFileHTTP(h serving.Handler) bool {
	return false
}

// ServeNotFoundHTTP responds with 404
func (s *Serverless) ServeNotFoundHTTP(h serving.Handler) {
	httperrors.Serve404(h.Writer)
}

// New returns a new serving instance
func New(cluster Cluster) serving.Serving {
	return &Serverless{}
}
