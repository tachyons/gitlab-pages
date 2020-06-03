package handlers

import (
	"net/http"

	"gitlab.com/gitlab-org/gitlab-pages/internal"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
)

// Handlers take care of handling specific requests
type Handlers struct {
	Auth     internal.Auth
	Artifact internal.Artifact
}

// New when provided the arguments defined herein, returns a pointer to an
// Handlers that is used to handle requests.
func New(auth internal.Auth, artifact internal.Artifact) *Handlers {
	return &Handlers{
		Auth:     auth,
		Artifact: artifact,
	}
}

func (a *Handlers) checkIfLoginRequiredOrInvalidToken(w http.ResponseWriter, r *http.Request, token string) func(*http.Response) bool {
	return func(resp *http.Response) bool {
		if resp.StatusCode == http.StatusNotFound {
			if token == "" {
				if !a.Auth.IsAuthSupported() {
					// Auth is not supported, probably means no access or does not exist but we cannot try with auth
					return false
				}

				logging.LogRequest(r).Debug("Artifact API response was 404 without token, try with authentication")

				// Authenticate user
				if a.Auth.RequireAuth(w, r) {
					return true
				}
			} else {
				logging.LogRequest(r).Debug("Artifact API response was 404 with authentication")
			}
		}

		if a.Auth.CheckResponseForInvalidToken(w, r, resp) {
			return true
		}

		return false
	}
}

// HandleArtifactRequest handles all artifact related requests, will return true if request was handled here
func (a *Handlers) HandleArtifactRequest(host string, w http.ResponseWriter, r *http.Request) bool {
	// In the event a host is prefixed with the artifact prefix an artifact
	// value is created, and an attempt to proxy the request is made

	// Always try to add token to the request if it exists
	token, err := a.Auth.GetTokenIfExists(w, r)
	if err != nil {
		return true
	}

	// nolint: bodyclose
	// a.checkIfLoginRequiredOrInvalidToken returns a response.Body, closing this body is responsibility
	// of the TryMakeRequest implementation
	return a.Artifact.TryMakeRequest(host, w, r, token, a.checkIfLoginRequiredOrInvalidToken(w, r, token))
}
