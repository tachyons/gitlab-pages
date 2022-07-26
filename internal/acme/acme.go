package acme

import (
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/host"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
)

// FallbackStrategy try to solve the acme challenge before redirecting to GitLab
type FallbackStrategy func(http.ResponseWriter, *http.Request) bool

// ServeAcmeChallenges identifies if request is acme-challenge and redirects to GitLab in that case
func ServeAcmeChallenges(w http.ResponseWriter, r *http.Request, fallback FallbackStrategy, gitlabURL *url.URL) bool {
	if !IsAcmeChallenge(r.URL.Path) {
		return false
	}

	if fallback(w, r) {
		return true
	}

	redirectToGitlab(w, r, gitlabURL)
	return true
}

func IsAcmeChallenge(path string) bool {
	return strings.HasPrefix(filepath.Clean(path), "/.well-known/acme-challenge/")
}

func redirectToGitlab(w http.ResponseWriter, r *http.Request, gitlabURL *url.URL) {
	redirectURL := *gitlabURL

	redirectURL.Path = "/-/acme-challenge"
	query := redirectURL.Query()
	query.Set("domain", host.FromRequest(r))
	query.Set("token", filepath.Base(r.URL.Path))
	redirectURL.RawQuery = query.Encode()

	logging.LogRequest(r).WithField("redirect_url", redirectURL).Debug("Redirecting to GitLab for processing acme challenge")

	http.Redirect(w, r, redirectURL.String(), http.StatusTemporaryRedirect)
}
