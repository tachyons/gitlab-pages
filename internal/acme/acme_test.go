package acme

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

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
	middlewareMalformed = &Middleware{GitlabURL: ":foo"}
)

func TestServeAcmeChallengesNotConfigured(t *testing.T) {
	require.HTTPStatusCode(t, serveAcmeOrNotFound(nil, domain), http.MethodGet, challengeURL, nil, http.StatusNotFound)
}

func TestServeAcmeChallengeMalformed(t *testing.T) {
	require.HTTPStatusCode(t, serveAcmeOrNotFound(middlewareMalformed, domain), http.MethodGet, challengeURL, nil, http.StatusNotFound)
}

func TestServeAcmeChallengeWhenPresent(t *testing.T) {
	require.HTTPStatusCode(t, serveAcmeOrNotFound(middleware, domainWithChallenge), http.MethodGet, challengeURL, nil, http.StatusNotFound)
}

func TestServeAcmeChallengeWhenMissing(t *testing.T) {
	testhelpers.AssertRedirectTo(
		t, serveAcmeOrNotFound(middleware, domain),
		"GET", challengeURL, nil,
		"https://gitlab.example.com/-/acme-challenge?domain=example.com&token=token",
	)

	require.HTTPStatusCode(t, serveAcmeOrNotFound(middleware, domain), http.MethodGet, indexURL, nil, http.StatusNotFound)
}
