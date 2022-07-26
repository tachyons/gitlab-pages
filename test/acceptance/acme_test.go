package acceptance_test

import (
	"io"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

const (
	existingAcmeTokenPath    = "/.well-known/acme-challenge/existingtoken"
	notExistingAcmeTokenPath = "/.well-known/acme-challenge/notexistingtoken"
)

func TestAcmeChallengesWhenItIsNotConfigured(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
	)

	tests := map[string]struct {
		token           string
		expectedStatus  int
		expectedContent string
	}{
		"When domain folder contains requested acme challenge it responds with it": {
			token:           existingAcmeTokenPath,
			expectedStatus:  http.StatusOK,
			expectedContent: "this is token\n",
		},
		"When domain folder does not contain requested acme challenge it returns 404": {
			token:           notExistingAcmeTokenPath,
			expectedStatus:  http.StatusNotFound,
			expectedContent: "The page you're looking for could not be found.",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
				test.token)

			require.NoError(t, err)
			require.Equal(t, test.expectedStatus, rsp.StatusCode)
			body, err := io.ReadAll(rsp.Body)
			require.NoError(t, rsp.Body.Close())
			require.NoError(t, err)

			require.Contains(t, string(body), test.expectedContent)
		})
	}
}

func TestAcmeChallengesWhenItIsConfigured(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		withExtraArgument("gitlab-server", "https://gitlab-acme.com"),
	)

	tests := map[string]struct {
		token            string
		expectedStatus   int
		expectedContent  string
		expectedLocation string
	}{
		"When domain folder contains requested acme challenge it responds with it": {
			token:           existingAcmeTokenPath,
			expectedStatus:  http.StatusOK,
			expectedContent: "this is token\n",
		},
		"When domain folder doesn't contains requested acme challenge it redirects to GitLab": {
			token:            notExistingAcmeTokenPath,
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedContent:  "",
			expectedLocation: "https://gitlab-acme.com/-/acme-challenge?domain=withacmechallenge.domain.com&token=notexistingtoken",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
				test.token)

			require.NoError(t, err)
			require.Equal(t, test.expectedStatus, rsp.StatusCode)
			body, err := io.ReadAll(rsp.Body)
			require.NoError(t, rsp.Body.Close())
			require.NoError(t, err)

			require.Contains(t, string(body), test.expectedContent)

			redirectURL, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			require.Equal(t, redirectURL.String(), test.expectedLocation)
		})
	}
}

func TestAcmeChallengesNoRedirection(t *testing.T) {
	RunPagesProcess(t,
		withListeners([]ListenSpec{httpListener}),
		withExtraArgument("gitlab-server", "https://gitlab-acme.com"),
	)

	tests := map[string]struct {
		path             string
		expectedStatus   int
		expectedContent  string
		expectedLocation string
	}{
		"wildcard redirect should not redirect acme challenge": {
			path:            existingAcmeTokenPath,
			expectedStatus:  http.StatusOK,
			expectedContent: "this is token\n",
		},
		"non-acme paths should be redirected": {
			path: "/example",
			// rule inside _redirects is a 200 rewrite of /index.html
			expectedStatus:  http.StatusOK,
			expectedContent: "acme-challenge-project\n",
		},
		"When domain folder doesn't contain requested acme challenge it redirects to GitLab": {
			path:             notExistingAcmeTokenPath,
			expectedStatus:   http.StatusTemporaryRedirect,
			expectedContent:  "",
			expectedLocation: "https://gitlab-acme.com/-/acme-challenge?domain=acmewithredirects.domain.com&token=notexistingtoken",
		},
	}

	for tn, tt := range tests {
		t.Run(tn, func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, "acmewithredirects.domain.com", tt.path)
			require.NoError(t, err)
			testhelpers.Close(t, rsp.Body)

			require.Equal(t, tt.expectedStatus, rsp.StatusCode)
			body, err := io.ReadAll(rsp.Body)
			require.NoError(t, rsp.Body.Close())
			require.NoError(t, err)

			require.Contains(t, string(body), tt.expectedContent)

			redirectURL, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			require.Equal(t, redirectURL.String(), tt.expectedLocation)
		})
	}
}
