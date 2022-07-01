package redirects

import (
	"testing"

	"github.com/stretchr/testify/require"
	netlifyRedirects "github.com/tj/go-redirects"

	"gitlab.com/gitlab-org/gitlab-pages/internal/feature"
)

type testCaseData struct {
	rule         string
	path         string
	expectMatch  bool
	expectedPath string
}

type matchingTestSuite = map[string]testCaseData

func mergeTestSuites(suites ...matchingTestSuite) matchingTestSuite {
	var merged = make(matchingTestSuite)
	for _, suite := range suites {
		for name, test := range suite {
			merged[name] = test
		}
	}

	return merged
}

var testsWithoutPlaceholders = map[string]testCaseData{
	"exact_match": {
		rule:         "/foo/ /bar/",
		path:         "/foo/",
		expectMatch:  true,
		expectedPath: "/bar/",
	},
	"single_trailing_slash": {
		rule:         "/foo/ /bar/",
		path:         "/foo",
		expectMatch:  true,
		expectedPath: "/bar/",
	},
	"ignore_missing_slash": {
		rule:         "/foo /bar/",
		path:         "/foo/",
		expectMatch:  true,
		expectedPath: "/bar/",
	},
	"no_match": {
		rule:         "/foo /bar/",
		path:         "/foo/bar",
		expectMatch:  false,
		expectedPath: "",
	},
}

func Test_matchesRule(t *testing.T) {
	t.Setenv(feature.RedirectsPlaceholders.EnvVariable, "true")

	tests := mergeTestSuites(testsWithoutPlaceholders, map[string]testCaseData{
		// Note: the following 3 cases behave differently when
		// placeholders are disabled. See the similar test cases below.
		"multiple_trailing_slashes": {
			rule:         "/foo/ /bar/",
			path:         "/foo//",
			expectMatch:  true,
			expectedPath: "/bar/",
		},
		"multiple_leading_slashes": {
			rule:         "/foo/ /bar/",
			path:         "//foo",
			expectMatch:  true,
			expectedPath: "/bar/",
		},
		"multiple_slashes_in_middle": {
			rule:         "/foo/bar /baz/",
			path:         "/foo//bar",
			expectMatch:  true,
			expectedPath: "/baz/",
		},

		"splat_match": {
			rule:         "/foo/*/bar /foo/:splat/qux",
			path:         "/foo/baz/bar",
			expectMatch:  true,
			expectedPath: "/foo/baz/qux",
		},
		"splat_match_multiple_segments": {
			rule:         "/foo/*/bar /foo/:splat/qux",
			path:         "/foo/hello/world/bar",
			expectMatch:  true,
			expectedPath: "/foo/hello/world/qux",
		},
		"splat_match_ignore_trailing_slash": {
			rule:         "/foo/*/bar /foo/:splat/qux",
			path:         "/foo/baz/bar/",
			expectMatch:  true,
			expectedPath: "/foo/baz/qux",
		},
		"splat_match_end": {
			rule:         "/foo/* /qux/:splat",
			path:         "/foo/baz/bar",
			expectMatch:  true,
			expectedPath: "/qux/baz/bar",
		},
		"splat_match_end_with_slash": {
			rule:         "/foo/* /qux/:splat",
			path:         "/foo/baz/bar/",
			expectMatch:  true,
			expectedPath: "/qux/baz/bar/",
		},
		"splat_match_beginning": {
			rule:         "/*/baz/bar /qux/:splat",
			path:         "/foo/baz/bar",
			expectMatch:  true,
			expectedPath: "/qux/foo",
		},
		"splat_match_empty_suffix": {
			rule:         "/foo/* /qux/:splat",
			path:         "/foo/",
			expectMatch:  true,
			expectedPath: "/qux/",
		},
		"splat_consumes_trailing_slash": {
			rule:         "/foo/* /qux/:splat",
			path:         "/foo",
			expectMatch:  true,
			expectedPath: "/qux/",
		},
		"splat_match_empty_prefix": {
			rule:         "/*/foo /qux/:splat",
			path:         "/foo",
			expectMatch:  true,
			expectedPath: "/qux/",
		},
		"splat_mid_segment": {
			rule:         "/foo*bar /qux/:splat",
			path:         "/foobazbar",
			expectMatch:  false,
			expectedPath: "",
		},
		"splat_mid_segment_no_content": {
			rule:         "/foo*bar /qux/:splat",
			path:         "/foobar",
			expectMatch:  false,
			expectedPath: "",
		},
		"lone_splat": {
			rule:         "/* /qux/:splat",
			path:         "/foo/bar",
			expectMatch:  true,
			expectedPath: "/qux/foo/bar",
		},
		"multiple_splats": {
			rule:         "/foo/*/bar/*/baz /qux/:splat",
			path:         "/foo/hello/bar/world/baz",
			expectMatch:  true,
			expectedPath: "/qux/hello",
		},
		"duplicate_splat_replacements": {
			rule:         "/foo/* /qux/:splat/:splat",
			path:         "/foo/hello",
			expectMatch:  true,
			expectedPath: "/qux/hello/hello",
		},
		"splat_missing_path_segment_behavior": {
			rule:         "/foo/*/bar /foo/:splat/qux",
			path:         "/foo/bar",
			expectMatch:  true,
			expectedPath: "/foo/qux",
		},
		"missing_splat_placeholder": {
			rule:         "/foo/ /qux/:splat",
			path:         "/foo/",
			expectMatch:  true,
			expectedPath: "/qux/",
		},
		"placeholder_match": {
			rule:         "/foo/:year/:month/:day/bar /qux/:year-:month-:day",
			path:         "/foo/2021/08/16/bar",
			expectMatch:  true,
			expectedPath: "/qux/2021-08-16",
		},
		"placeholder_match_end": {
			rule:         "/foo/:placeholder /qux/:placeholder",
			path:         "/foo/bar",
			expectMatch:  true,
			expectedPath: "/qux/bar",
		},
		"placeholder_match_beginning": {
			rule:         "/:placeholder/foo /qux/:placeholder",
			path:         "/baz/foo",
			expectMatch:  true,
			expectedPath: "/qux/baz",
		},
		"placeholder_no_multiple_segments": {
			rule:         "/foo/:placeholder/bar /foo/:placeholder/qux",
			path:         "/foo/hello/world/bar",
			expectMatch:  false,
			expectedPath: "",
		},
		"placeholder_at_beginning_no_content": {
			rule:         "/:placeholder/foo /qux/:placeholder",
			path:         "/foo",
			expectMatch:  false,
			expectedPath: "",
		},
		"placeholder_at_end_no_content": {
			rule:         "/foo/:placeholder /qux/:placeholder",
			path:         "/foo/",
			expectMatch:  false,
			expectedPath: "",
		},
		"placeholder_mid_segment_in_from": {
			rule:         "/foo:placeholder /qux/:placeholder",
			path:         "/foorbar",
			expectMatch:  false,
			expectedPath: "",
		},
		"placeholder_mid_segment_in_to": {
			rule:         "/foo/:placeholder /qux/bar:placeholder",
			path:         "/foo/baz",
			expectMatch:  true,
			expectedPath: "/qux/barbaz",
		},
		"placeholder_missing_replacement_with_substring": {
			rule:         "/:foo /:foobar",
			path:         "/baz",
			expectMatch:  true,
			expectedPath: "/",
		},
		"placeholder_mid_segment_no_content": {
			rule:         "/foo:placeholder /qux/:splat",
			path:         "/foo",
			expectMatch:  false,
			expectedPath: "",
		},
		"placeholder_name_substring": {
			rule:         "/foo/:foo/:foobar /qux/:foo/:foobar",
			path:         "/foo/baz/quux",
			expectMatch:  true,
			expectedPath: "/qux/baz/quux",
		},
		"lone_placeholder": {
			rule:         "/:placeholder /qux/:placeholder",
			path:         "/foo",
			expectMatch:  true,
			expectedPath: "/qux/foo",
		},
		"duplicate_placeholders": {
			rule:         "/foo/:placeholder/bar/:placeholder/baz /qux/:placeholder",
			path:         "/foo/hello/bar/world/baz",
			expectMatch:  true,
			expectedPath: "/qux/hello",
		},
		"duplicate_placeholder_replacements": {
			rule:         "/foo/:placeholder /qux/:placeholder/:placeholder",
			path:         "/foo/hello",
			expectMatch:  true,
			expectedPath: "/qux/hello/hello",
		},
		"splat_and_placeholder_named_splat": {
			rule:         "/foo/*/bar/:splat /qux/:splat",
			path:         "/foo/hello/bar/world",
			expectMatch:  true,
			expectedPath: "/qux/hello",
		},
		"placeholder_named_splat_and_splat": {
			rule:         "/foo/:splat/bar/* /qux/:splat",
			path:         "/foo/hello/bar/world",
			expectMatch:  true,
			expectedPath: "/qux/hello",
		},

		// Note: These differ slightly from Netlify's matching behavior.
		// GitLab replaces _all_ placeholders in the "to" path, even if
		// the placeholder doesn't have corresponding match in the "from".
		// Netlify only replaces placeholders that appear in the "from".
		"missing_placeholder_exact_match": {
			rule:        "/foo/ /qux/:placeholder",
			path:        "/foo/",
			expectMatch: true,

			// Netlify would instead redirect to "/qux/:placeholder"
			expectedPath: "/qux/",
		},
		"missing_placeholder_nonexact_match": {
			rule:        "/foo/:placeholderA /qux/:placeholderB",
			path:        "/foo/bar",
			expectMatch: true,

			// Netlify would instead redirect to "/qux/:placeholderB"
			expectedPath: "/qux/",
		},
	})

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rules, err := netlifyRedirects.ParseString(tt.rule)
			require.NoError(t, err)

			isMatch, path := matchesRule(&rules[0], tt.path)
			require.Equal(t, tt.expectMatch, isMatch)
			require.Equal(t, tt.expectedPath, path)
		})
	}
}

// Tests matching behavior when the `FF_ENABLE_PLACEHOLDERS`
// feature flag is not enabled. These tests can be removed when the
// `FF_ENABLE_PLACEHOLDERS` flag is removed.
func Test_matchesRule_NoPlaceholders(t *testing.T) {
	// disable placeholders on purpose
	t.Setenv(feature.RedirectsPlaceholders.EnvVariable, "false")

	tests := mergeTestSuites(testsWithoutPlaceholders, map[string]testCaseData{
		// Note: the following 3 case behaves differently when
		// placeholders are enabled. See the similar test cases above.
		"multiple_trailing_slashes": {
			rule:         "/foo/ /bar/",
			path:         "/foo//",
			expectMatch:  false,
			expectedPath: "",
		},
		"multiple_leading_slashes": {
			rule:         "/foo/ /bar/",
			path:         "//foo",
			expectMatch:  false,
			expectedPath: "",
		},
		"multiple_slashes_in_middle": {
			rule:         "/foo/bar /baz/",
			path:         "/foo//bar",
			expectMatch:  false,
			expectedPath: "",
		},
	})

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			rules, err := netlifyRedirects.ParseString(tt.rule)
			require.NoError(t, err)

			isMatch, path := matchesRule(&rules[0], tt.path)
			require.Equal(t, tt.expectMatch, isMatch)
			require.Equal(t, tt.expectedPath, path)
		})
	}
}
