package fileresolver

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveFilePath(t *testing.T) {
	tests := []struct {
		name             string
		lookupPath       string
		subPath          string
		urlPath          string
		expectedFullPath string
		expectedErr      error
	}{
		{
			name:        "file_does_not_exist",
			lookupPath:  "../../../shared/pages/group/group.no.projects/",
			urlPath:     "/group.no.projects/",
			expectedErr: errFileNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath, err := ResolveFilePath(tt.lookupPath, tt.subPath, tt.urlPath)
			require.Equal(t, tt.expectedFullPath, fullPath)
			require.Equal(t, tt.expectedErr, err)
		})
	}
}
