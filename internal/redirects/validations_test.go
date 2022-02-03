package redirects

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	netlifyRedirects "github.com/tj/go-redirects"
)

func TestRedirectsValidateUrl(t *testing.T) {
	enablePlaceholders(t)

	tests := map[string]struct {
		url         string
		expectedErr error
	}{
		"valid_url": {
			url: "/goto.html",
		},
		"no_domain_level_redirects": {
			url:         "https://GitLab.com",
			expectedErr: errNoDomainLevelRedirects,
		},
		"no_schemaless_url_domain_level_redirects": {
			url:         "//GitLab.com/pages.html",
			expectedErr: errNoDomainLevelRedirects,
		},
		"no_bare_domain_level_redirects": {
			url:         "GitLab.com",
			expectedErr: errNoStartingForwardSlashInURLPath,
		},
		"no_parent_traversing_relative_url": {
			url:         "../target.html",
			expectedErr: errNoStartingForwardSlashInURLPath,
		},
		"too_many_slashes": {
			url:         strings.Repeat("/a", 26),
			expectedErr: errTooManyPathSegments,
		},
		"placeholders": {
			url: "/news/:year/:month/:date/:slug",
		},
		"splats": {
			url: "/blog/*",
		},
		"splat_placeholders": {
			url: "/new/path/:splat",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateURL(tt.url)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

// Tests validation rules that only apply when the `FF_ENABLE_PLACEHOLDERS`
// feature flag is not enabled. These tests can be removed when the
// `FF_ENABLE_PLACEHOLDERS` flag is removed.
func TestRedirectsValidateUrlNoPlaceholders(t *testing.T) {
	tests := map[string]struct {
		url         string
		expectedErr error
	}{
		"no_splats": {
			url:         "/blog/*",
			expectedErr: errNoSplats,
		},
		"no_placeholders": {
			url:         "/news/:year/:month/:date/:slug",
			expectedErr: errNoPlaceholders,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateURL(tt.url)
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}

func TestRedirectsValidateRule(t *testing.T) {
	enablePlaceholders(t)

	tests := map[string]struct {
		rule        string
		expectedErr error
	}{
		"valid_rule": {
			rule: "/goto.html /target.html 301",
		},
		"invalid_from_url": {
			rule:        "invalid.com /teapot.html 302",
			expectedErr: errNoStartingForwardSlashInURLPath,
		},
		"invalid_to_url": {
			rule:        "/goto.html invalid.com",
			expectedErr: errNoStartingForwardSlashInURLPath,
		},
		"no_parameters": {
			rule: "/	/something	302	foo=bar",
			expectedErr: errNoParams,
		},
		"invalid_status": {
			rule:        "/goto.html /target.html 418",
			expectedErr: errUnsupportedStatus,
		},
		"force_not_supported": {
			rule:        "/goto.html /target.html 302!",
			expectedErr: errNoForce,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rules, err := netlifyRedirects.ParseString(tt.rule)
			require.NoError(t, err)

			err = validateRule(rules[0])
			require.ErrorIs(t, err, tt.expectedErr)
		})
	}
}
