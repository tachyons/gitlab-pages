package deploy

import (
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExtractZip(t *testing.T) {
	testCases := []struct {
		desc         string
		archive      string
		mustExist    map[string]string
		mustNotExist []string
		matchError   func(error) bool
	}{
		{
			desc:    "archive with file inside and outside public/",
			archive: "testdata/test1.zip",
			mustExist: map[string]string{
				"public/h/e/l/l/o": "world\n",
			},
			mustNotExist: []string{"foo"},
		},
		{
			desc:    "archive with no file in public/",
			archive: "testdata/test2.zip",
			matchError: func(err error) bool {
				return err == ErrNoPublicFiles
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			tmpDir, err := ioutil.TempDir("", "gitlab-pages")
			require.NoError(t, err)
			defer os.RemoveAll(tmpDir)

			extractError := ExtractZip(tc.archive, tmpDir)
			if tc.matchError != nil {
				require.True(t, tc.matchError(extractError), "error %v does not match", extractError)
				return
			}

			require.NoError(t, extractError)

			for file, content := range tc.mustExist {
				actualContent, err := ioutil.ReadFile(path.Join(tmpDir, file))
				require.NoError(t, err, "read %q", file)
				require.Equal(t, content, string(actualContent), "content of %q", file)
			}

			for _, file := range tc.mustNotExist {
				_, err = os.Stat(path.Join(tmpDir, file))
				require.True(t, os.IsNotExist(err), "error should be '%q does not exist': %v", file, err)
			}
		})
	}
}
