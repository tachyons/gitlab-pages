package redirects

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_normalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "add_trailing_slash",
			path:     "foo",
			expected: "foo/",
		},
		{
			name:     "leave_existing_trailing_slash",
			path:     "foo/",
			expected: "foo/",
		},
		{
			name:     "leave_existing_double_trailing_slash",
			path:     "foo//",
			expected: "foo//",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := normalizePath(tt.path)
			require.Equal(t, tt.expected, got)
		})
	}
}
