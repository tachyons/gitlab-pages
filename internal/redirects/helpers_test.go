package redirects

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_normalizePath(t *testing.T) {
	tests := map[string]struct {
		name     string
		path     string
		expected string
	}{
		"add_trailing_slash": {
			path:     "foo",
			expected: "foo/",
		},
		"leave_existing_trailing_slash": {
			path:     "foo/",
			expected: "foo/",
		},
		"leave_existing_double_trailing_slash": {
			path:     "foo//",
			expected: "foo//",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got := normalizePath(tt.path)
			require.Equal(t, tt.expected, got)
		})
	}
}
