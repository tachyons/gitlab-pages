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
		expectedErr string
	}{
		"valid_url": {
			url:         "/goto.html",
			expectedErr: "",
		},
		"no_domain_level_redirects": {
			url:         "https://GitLab.com",
			expectedErr: errNoDomainLevelRedirects.Error(),
		},
		"no_schemaless_url_domain_level_redirects": {
			url:         "//GitLab.com/pages.html",
			expectedErr: errNoDomainLevelRedirects.Error(),
		},
		"no_bare_domain_level_redirects": {
			url:         "GitLab.com",
			expectedErr: errNoStartingForwardSlashInURLPath.Error(),
		},
		"no_parent_traversing_relative_url": {
			url:         "../target.html",
			expectedErr: errNoStartingForwardSlashInURLPath.Error(),
		},
		"too_many_slashes": {
			url:         strings.Repeat("/a", 26),
			expectedErr: errTooManyPathSegments.Error(),
		},
		"placeholders": {
			url:         "/news/:year/:month/:date/:slug",
			expectedErr: "",
		},
		"splats": {
			url:         "/blog/*",
			expectedErr: "",
		},
		"splat_placeholders": {
			url:         "/new/path/:splat",
			expectedErr: "",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateURL(tt.url)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

// Tests validation rules that only apply when the `FF_ENABLE_PLACEHOLDERS`
// feature flag is not enabled. These tests can be removed when the
// `FF_ENABLE_PLACEHOLDERS` flag is removed.
func TestRedirectsValidateUrlNoPlaceholders(t *testing.T) {
	tests := map[string]struct {
		url         string
		expectedErr string
	}{
		"no_splats": {
			url:         "/blog/*",
			expectedErr: errNoSplats.Error(),
		},
		"no_placeholders": {
			url:         "/news/:year/:month/:date/:slug",
			expectedErr: errNoPlaceholders.Error(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := validateURL(tt.url)
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}

func TestRedirectsValidateRule(t *testing.T) {
	enablePlaceholders(t)

	tests := map[string]struct {
		rule        string
		expectedErr string
	}{
		"valid_rule": {
			rule:        "/goto.html /target.html 301",
			expectedErr: "",
		},
		"invalid_from_url": {
			rule:        "invalid.com /teapot.html 302",
			expectedErr: errNoStartingForwardSlashInURLPath.Error(),
		},
		"invalid_to_url": {
			rule:        "/goto.html invalid.com",
			expectedErr: errNoStartingForwardSlashInURLPath.Error(),
		},
		"no_parameters": {
			rule: "/	/something	302	foo=bar",
			expectedErr: errNoParams.Error(),
		},
		"invalid_status": {
			rule:        "/goto.html /target.html 418",
			expectedErr: errUnsupportedStatus.Error(),
		},
		"force_not_supported": {
			rule:        "/goto.html /target.html 302!",
			expectedErr: errNoForce.Error(),
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rules, err := netlifyRedirects.ParseString(tt.rule)
			require.NoError(t, err)

			err = validateRule(rules[0])
			if tt.expectedErr != "" {
				require.EqualError(t, err, tt.expectedErr)
				return
			}

			require.NoError(t, err)
		})
	}
}
