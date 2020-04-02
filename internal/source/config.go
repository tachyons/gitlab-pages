package source

import "gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"

// Config represents an interface that is configuration provider for client
// capable of communicating with GitLab
type Config client.Config
