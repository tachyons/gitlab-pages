package redirects

import (
	"context"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	netlifyRedirects "github.com/tj/go-redirects"

	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestRedirectsRewrite(t *testing.T) {
	t.Setenv(feature.RedirectsPlaceholders.EnvVariable, "true")

	tests := []struct {
		name           string
		url            string
		rule           string
		expectedURL    string
		expectedStatus int
		expectedErr    error
	}{
		{
			name:           "No rules given",
			url:            "/no-redirect/",
			rule:           "",
			expectedURL:    "",
			expectedStatus: 0,
			expectedErr:    ErrNoRedirect,
		},
		{
			name:           "No matching rules",
			url:            "/no-redirect/",
			rule:           "/cake-portal.html  /still-alive.html 301",
			expectedURL:    "",
			expectedStatus: 0,
			expectedErr:    ErrNoRedirect,
		},
		{
			name:           "Matching rule redirects",
			url:            "/cake-portal.html",
			rule:           "/cake-portal.html  /still-alive.html 301",
			expectedURL:    "/still-alive.html",
			expectedStatus: http.StatusMovedPermanently,
		},
		{
			name:           "Does not redirect to invalid rule",
			url:            "/goto.html",
			rule:           "/goto.html GitLab.com 301",
			expectedURL:    "",
			expectedStatus: 0,
			expectedErr:    ErrNoRedirect,
		},
		{
			name:           "Matches trailing slash rule to no trailing slash URL",
			url:            "/cake-portal",
			rule:           "/cake-portal/  /still-alive/ 301",
			expectedURL:    "/still-alive/",
			expectedStatus: http.StatusMovedPermanently,
		},
		{
			name:           "Matches trailing slash rule to trailing slash URL",
			url:            "/cake-portal/",
			rule:           "/cake-portal/  /still-alive/ 301",
			expectedURL:    "/still-alive/",
			expectedStatus: http.StatusMovedPermanently,
		},
		{
			name:           "Matches no trailing slash rule to no trailing slash URL",
			url:            "/cake-portal",
			rule:           "/cake-portal  /still-alive 301",
			expectedURL:    "/still-alive",
			expectedStatus: http.StatusMovedPermanently,
		},
		{
			name:           "Matches no trailing slash rule to trailing slash URL",
			url:            "/cake-portal/",
			rule:           "/cake-portal  /still-alive 301",
			expectedURL:    "/still-alive",
			expectedStatus: http.StatusMovedPermanently,
		},
		{
			name:           "matches_splat_rule",
			url:            "/the-cake/is-delicious",
			rule:           "/the-cake/* /is-a-lie 200",
			expectedURL:    "/is-a-lie",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "replaces_splat_placeholdes",
			url:            "/from/weighted/companion/cube/path",
			rule:           "/from/*/path /to/:splat/path 200",
			expectedURL:    "/to/weighted/companion/cube/path",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "matches_placeholder_rule",
			url:            "/the/cake/is/delicious",
			rule:           "/the/:placeholder/is/delicious /the/:placeholder/is/a/lie 200",
			expectedURL:    "/the/cake/is/a/lie",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "does_not_redirect_acme_challenges",
			url:            "/.well-known/acme-challenge/token",
			rule:           "/* /to/path 200",
			expectedURL:    "",
			expectedStatus: 0,
			expectedErr:    ErrNoRedirect,
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
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestRedirectsParseRedirects(t *testing.T) {
	ctx := context.Background()

	root, tmpDir := testhelpers.TmpDir(t)

	tests := []struct {
		name          string
		redirectsFile string
		expectedRules int
		expectedErr   error
	}{
		{
			name:          "No `_redirects` file present",
			redirectsFile: "",
			expectedRules: 0,
			expectedErr:   errConfigNotFound,
		},
		{
			name:          "Everything working as expected",
			redirectsFile: `/goto.html /target.html 301`,
			expectedRules: 1,
		},
		{
			name:          "Invalid _redirects syntax gives no rules",
			redirectsFile: `foobar::baz`,
			expectedRules: 0,
		},
		{
			name:          "Config file too big",
			redirectsFile: strings.Repeat("a", 2*cfg.MaxConfigSize),
			expectedRules: 0,
			expectedErr:   errFileTooLarge,
		},
		// In future versions of `github.com/tj/go-redirects`,
		// this may not throw a parsing error and this test could be removed
		{
			name:          "Parsing error is caught",
			redirectsFile: "/store id=:id  /blog/:id  301",
			expectedRules: 0,
			expectedErr:   errFailedToParseConfig,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.redirectsFile != "" {
				err := os.WriteFile(path.Join(tmpDir, ConfigFile), []byte(tt.redirectsFile), 0600)
				require.NoError(t, err)
			}

			redirects := ParseRedirects(ctx, root)

			require.ErrorIs(t, redirects.error, tt.expectedErr)
			require.Len(t, redirects.rules, tt.expectedRules)
		})
	}
}

func TestMaxRuleCount(t *testing.T) {
	root, tmpDir := testhelpers.TmpDir(t)

	err := os.WriteFile(path.Join(tmpDir, ConfigFile), []byte(strings.Repeat("/goto.html /target.html 301\n", cfg.MaxRuleCount-1)+
		"/1000.html /target1000 301\n"+
		"/1001.html /target1001 301\n",
	), 0600)
	require.NoError(t, err)

	redirects := ParseRedirects(context.Background(), root)

	testFn := func(path, expectedToURL string, expectedStatus int, expectedErr error) func(t *testing.T) {
		return func(t *testing.T) {
			originalURL, err := url.Parse(path)
			require.NoError(t, err)

			toURL, status, err := redirects.Rewrite(originalURL)
			if expectedErr != nil {
				require.ErrorIs(t, err, expectedErr)
				return
			}

			require.NoError(t, err)

			require.Equal(t, expectedToURL, toURL.String())
			require.Equal(t, expectedStatus, status)
		}
	}

	t.Run("maxRuleCount matches", testFn("/1000.html", "/target1000", http.StatusMovedPermanently, nil))
	t.Run("maxRuleCount+1 does not match", testFn("/1001.html", "", 0, ErrNoRedirect))
}
