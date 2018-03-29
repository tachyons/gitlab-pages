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
		{
			desc:    "archive with evil symlink",
			archive: "testdata/test3.zip",
			mustExist: map[string]string{
				// The test3.zip archive contains a symlink to /etc/passwd. This test
				// asserts that instead of that symlink, we get a regular file whose
				// contents are "/etc/passwd". If the extracted "public/passwd" was an
				// actual symlink we would get the contents of the /etc/passwd file of
				// the system where the test runs.
				"public/passwd": "/etc/passwd",
				// The "public/bar" symlink tries to point to "foo" but we don't support
				// symlinks at the moment. Instead it creates a regular file with
				// contents "bar". TODO: support valid symlinks?
				"public/bar": "foo",
				// "foo" is a regular file with contents "not-bar"
				"public/foo": "not-bar\n",
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
