package fileresolver

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveFilePath(t *testing.T) {
	tests := []struct {
		name             string
		evalSymlinkFunc  evalSymlinkFunc
		lookupPath       string
		subPath          string
		urlPath          string
		expectedFullPath string
		expectedErr      error
	}{
		{
			name:             "file_exists_with_subpath_and_extension",
			evalSymlinkFunc:  func(in string) (string, error) { return in, nil },
			lookupPath:       "../../../shared/pages/group/group.test.io/public/",
			subPath:          "index.html",
			urlPath:          "/index.html",
			expectedFullPath: "../../../shared/pages/group/group.test.io/public/index.html",
		},
		{
			name:             "file_exists_without_extension",
			evalSymlinkFunc:  func(in string) (string, error) { return in, nil },
			lookupPath:       "../../../shared/pages/group/group.test.io/public/",
			subPath:          "index",
			urlPath:          "/index",
			expectedFullPath: "../../../shared/pages/group/group.test.io/public/index.html",
		},
		{
			name:             "file_exists_without_subpath",
			evalSymlinkFunc:  filepath.EvalSymlinks,
			lookupPath:       "../../../shared/pages/group/group.test.io/public/",
			subPath:          "",
			urlPath:          "/",
			expectedFullPath: "../../../shared/pages/group/group.test.io/public/index.html",
		},
		{
			name:            "file_does_not_exist",
			evalSymlinkFunc: filepath.EvalSymlinks,
			lookupPath:      "../../../shared/pages/group/group.no.projects/public/",
			subPath:         "unknown_file.html",
			urlPath:         "/group.no.projects/unknown_file.html",
			expectedErr:     errFileNotFound,
		},
		{
			name:             "symlink_inside_public",
			evalSymlinkFunc:  filepath.EvalSymlinks,
			lookupPath:       "../../../shared/pages/group/symlink/public/",
			subPath:          "index.html",
			urlPath:          "/symlink/index.html",
			expectedFullPath: "../../../shared/pages/group/symlink/public/content/index.html",
		},
		{
			name:            "symlink_outside_of_public_dir",
			evalSymlinkFunc: filepath.EvalSymlinks,
			lookupPath:      "../../../shared/pages/group/symlink/public/",
			subPath:         "outside.html",
			urlPath:         "/symlink/outside.html",
			expectedErr:     errFileNotInPublicDir,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fullPath, err := ResolveFilePath(tt.lookupPath, tt.subPath, tt.urlPath, tt.evalSymlinkFunc)
			require.Equal(t, tt.expectedErr, err)
			require.Equal(t, tt.expectedFullPath, fullPath)
		})
	}
}
