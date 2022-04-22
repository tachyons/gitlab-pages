package acme

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/host"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
)

// Middleware handles acme challenges by redirecting them to GitLab instance
type Middleware struct {
	GitlabURL string
}

// Domain interface represent D from domain package
type Domain interface {
	ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool
}

// ServeAcmeChallenges identifies if request is acme-challenge and redirects to GitLab in that case
func (m *Middleware) ServeAcmeChallenges(w http.ResponseWriter, r *http.Request, domain Domain) bool {
	if m == nil {
		return false
	}

	if !isAcmeChallenge(r.URL.Path) {
		return false
	}

	if domain.ServeFileHTTP(w, r) {
		return true
	}

	return m.redirectToGitlab(w, r)
}

func isAcmeChallenge(path string) bool {
	return strings.HasPrefix(filepath.Clean(path), "/.well-known/acme-challenge/")
}

func (m *Middleware) redirectToGitlab(w http.ResponseWriter, r *http.Request) bool {
	redirectURL, err := url.Parse(m.GitlabURL)
	if err != nil {
		logging.LogRequest(r).WithError(err).Error("Can't parse GitLab URL for acme challenge redirect")
		return false
	}

	redirectURL.Path = "/-/acme-challenge"
	query := redirectURL.Query()
	query.Set("domain", host.FromRequest(r))
	query.Set("token", filepath.Base(r.URL.Path))
	redirectURL.RawQuery = query.Encode()

	logging.LogRequest(r).WithField("redirect_url", redirectURL).Debug("Redirecting to GitLab for processing acme challenge")

	http.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
	return true
}
