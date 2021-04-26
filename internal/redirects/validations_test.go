package redirects

import (
	"testing"

	"github.com/stretchr/testify/require"
	netlifyRedirects "github.com/tj/go-redirects"
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
