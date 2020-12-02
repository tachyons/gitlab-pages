package serving

import "gitlab.com/gitlab-org/gitlab-pages/internal/config"

// Serving is an interface used to define a serving driver
type Serving interface {
	ServeFileHTTP(Handler) bool
	ServeNotFoundHTTP(Handler)
	Reconfigure(config *config.Config) error
}
