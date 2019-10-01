package acme

import (
	"net/http"
	"testing"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

type domainStub struct {
	hasAcmeChallenge bool
}

func (d *domainStub) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/.well-known/acme-challenge/token" {
		return d.hasAcmeChallenge
	}

	return false
}

func serveAcmeOrNotFound(m *Middleware, domain Domain) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !m.ServeAcmeChallenges(w, r, domain) {
			http.NotFound(w, r)
		}
	}
}

const (
	baseURL      = "http://example.com"
	indexURL     = baseURL + "/index.html"
	challengeURL = baseURL + "/.well-known/acme-challenge/token"
)

var (
	domainWithChallenge = &domainStub{hasAcmeChallenge: true}
	domain              = &domainStub{hasAcmeChallenge: false}
	middleware          = &Middleware{GitlabURL: "https://gitlab.example.com"}
)

func TestServeAcmeChallengesNotConfigured(t *testing.T) {
	testhelpers.AssertHTTP404(t, serveAcmeOrNotFound(nil, domain), "GET", challengeURL, nil, nil)
}

func TestServeAcmeChallengeWhenPresent(t *testing.T) {
	testhelpers.AssertHTTP404(t, serveAcmeOrNotFound(middleware, domainWithChallenge), "GET", challengeURL, nil, nil)
}

func TestServeAcmeChallengeWhenMissing(t *testing.T) {
	testhelpers.AssertRedirectTo(
		t, serveAcmeOrNotFound(middleware, domain),
		"GET", challengeURL, nil,
		"https://gitlab.example.com/-/acme-challenge?domain=example.com&token=token",
	)

	testhelpers.AssertHTTP404(t, serveAcmeOrNotFound(middleware, domain), "GET", indexURL, nil, nil)
}
