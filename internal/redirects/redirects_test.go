package redirects

import (
	"context"
	"io/ioutil"
	"net/url"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	netlifyRedirects "github.com/tj/go-redirects"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestRedirectsValidateUrl(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		expectedErr string
	}{
		{
			name:        "Valid url",
			url:         "/goto.html",
			expectedErr: "",
		},
		{
			name:        "No domain-level redirects",
			url:         "https://GitLab.com",
			expectedErr: errNoDomainLevelRedirects.Error(),
		},
		{
			name:        "No Schema-less URL domain-level redirects",
			url:         "//GitLab.com/pages.html",
			expectedErr: errNoDomainLevelRedirects.Error(),
		},
		{
			name:        "No bare domain-level redirects",
			url:         "GitLab.com",
			expectedErr: errNoStartingForwardSlashInURLPath.Error(),
		},
		{
			name:        "No parent traversing relative URL",
			url:         "../target.html",
			expectedErr: errNoStartingForwardSlashInURLPath.Error(),
		},
		{
			name:        "No splats",
			url:         "/blog/*",
			expectedErr: errNoSplats.Error(),
		},
		{
			name:        "No Placeholders",
			url:         "/news/:year/:month/:date/:slug",
			expectedErr: errNoPlaceholders.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateURL(tt.url)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRedirectsValidateRule(t *testing.T) {
	tests := []struct {
		name        string
		rule        string
		expectedErr string
	}{
		{
			name:        "valid rule",
			rule:        "/goto.html /target.html 301",
			expectedErr: "",
		},
		{
			name:        "invalid From URL",
			rule:        "invalid.com /teapot.html 302",
			expectedErr: errNoStartingForwardSlashInURLPath.Error(),
		},
		{
			name:        "invalid To URL",
			rule:        "/goto.html invalid.com",
			expectedErr: errNoStartingForwardSlashInURLPath.Error(),
		},
		{
			name: "No parameters",
			rule: "/	/something	302	foo=bar",
			expectedErr: errNoParams.Error(),
		},
		{
			name:        "Invalid status",
			rule:        "/goto.html /target.html 418",
			expectedErr: errUnsupportedStatus.Error(),
		},
		{
			name:        "Force not supported",
			rule:        "/goto.html /target.html 302!",
			expectedErr: errNoForce.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rules, err := netlifyRedirects.ParseString(tt.rule)
			require.NoError(t, err)

			err = validateRule(rules[0])
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRedirectsRewrite(t *testing.T) {
	tests := []struct {
		name           string
		url            string
		rule           string
		expectedURL    string
		expectedStatus int
		expectedErr    string
	}{
		{
			name:           "No rules given",
			url:            "/no-redirect/",
			rule:           "",
			expectedURL:    "",
			expectedStatus: 0,
			expectedErr:    ErrNoRedirect.Error(),
		},
		{
			name:           "No matching rules",
			url:            "/no-redirect/",
			rule:           "/cake-portal.html  /still-alive.html 301",
			expectedURL:    "",
			expectedStatus: 0,
			expectedErr:    ErrNoRedirect.Error(),
		},
		{
			name:           "Matching rule redirects",
			url:            "/cake-portal.html",
			rule:           "/cake-portal.html  /still-alive.html 301",
			expectedURL:    "/still-alive.html",
			expectedStatus: 301,
			expectedErr:    "",
		},
		{
			name:           "Does not redirect to invalid rule",
			url:            "/goto.html",
			rule:           "/goto.html GitLab.com 301",
			expectedURL:    "",
			expectedStatus: 0,
			expectedErr:    ErrNoRedirect.Error(),
		},
		{
			name:           "Matches trailing slash rule to no trailing slash URL",
			url:            "/cake-portal",
			rule:           "/cake-portal/  /still-alive/ 301",
			expectedURL:    "/still-alive/",
			expectedStatus: 301,
			expectedErr:    "",
		},
		{
			name:           "Matches trailing slash rule to trailing slash URL",
			url:            "/cake-portal/",
			rule:           "/cake-portal/  /still-alive/ 301",
			expectedURL:    "/still-alive/",
			expectedStatus: 301,
			expectedErr:    "",
		},
		{
			name:           "Matches no trailing slash rule to no trailing slash URL",
			url:            "/cake-portal",
			rule:           "/cake-portal  /still-alive 301",
			expectedURL:    "/still-alive",
			expectedStatus: 301,
			expectedErr:    "",
		},
		{
			name:           "Matches no trailing slash rule to trailing slash URL",
			url:            "/cake-portal/",
			rule:           "/cake-portal  /still-alive 301",
			expectedURL:    "/still-alive",
			expectedStatus: 301,
			expectedErr:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Redirects{}

			if tt.rule != "" {
				rules, err := netlifyRedirects.ParseString(tt.rule)
				require.NoError(t, err)
				r.rules = rules
			}

			url, err := url.Parse(tt.url)
			require.NoError(t, err)

			toURL, status, err := r.Rewrite(url)

			if tt.expectedURL != "" {
				require.Equal(t, tt.expectedURL, toURL.String())
			} else {
				require.Nil(t, toURL)
			}

			require.Equal(t, tt.expectedStatus, status)

			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestRedirectsParseRedirects(t *testing.T) {
	ctx := context.Background()

	root, tmpDir, cleanup := testhelpers.TmpDir(t, "ParseRedirects_tests")
	defer cleanup()

	tests := []struct {
		name          string
		redirectsFile string
		expectedRules int
		expectedErr   string
	}{
		{
			name:          "No `_redirects` file present",
			redirectsFile: "",
			expectedRules: 0,
			expectedErr:   errConfigNotFound.Error(),
		},
		{
			name:          "Everything working as expected",
			redirectsFile: `/goto.html /target.html 301`,
			expectedRules: 1,
			expectedErr:   "",
		},
		{
			name:          "Invalid _redirects syntax gives no rules",
			redirectsFile: `foobar::baz`,
			expectedRules: 0,
			expectedErr:   "",
		},
		{
			name:          "Config file too big",
			redirectsFile: strings.Repeat("a", 2*maxConfigSize),
			expectedRules: 0,
			expectedErr:   errFileTooLarge.Error(),
		},
		// In future versions of `github.com/tj/go-redirects`,
		// this may not throw a parsing error and this test could be removed
		{
			name:          "Parsing error is caught",
			redirectsFile: "/store id=:id  /blog/:id  301",
			expectedRules: 0,
			expectedErr:   errFailedToParseConfig.Error(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.redirectsFile != "" {
				err := ioutil.WriteFile(path.Join(tmpDir, ConfigFile), []byte(tt.redirectsFile), 0600)
				require.NoError(t, err)
			}

			redirects := ParseRedirects(ctx, root)

			if tt.expectedErr != "" {
				require.EqualError(t, redirects.error, tt.expectedErr)
			} else {
				require.NoError(t, redirects.error)
			}

			require.Len(t, redirects.rules, tt.expectedRules)
		})
	}
}
