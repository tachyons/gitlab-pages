package jail_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/jail"
)

func tmpJailPath() string {
	return path.Join(os.TempDir(), fmt.Sprintf("my-jail-%d", time.Now().Unix()))
}

func TestTimestampedJails(t *testing.T) {
	assert := assert.New(t)

	prefix := "jail"
	var mode os.FileMode = 0755

	j1 := jail.TimestampedJail(prefix, mode)
	j2 := jail.TimestampedJail(prefix, mode)

	assert.NotEqual(j1.Path, j2.Path())
}

func TestJailPath(t *testing.T) {
	assert := assert.New(t)

	jailPath := tmpJailPath()
	cage := jail.New(jailPath, 0755)

	assert.Equal(jailPath, cage.Path())
}

func TestJailBuild(t *testing.T) {
	assert := assert.New(t)

	jailPath := tmpJailPath()
	cage := jail.New(jailPath, 0755)

	_, err := os.Stat(cage.Path())
	assert.Error(err, "Jail path should not exist before Jail.Build()")

	err = cage.Build()
	assert.NoError(err)
	defer cage.Dispose()

	_, err = os.Stat(cage.Path())
	assert.NoError(err, "Jail path should exist after Jail.Build()")
}

func TestJailOnlySupportsOneBindMount(t *testing.T) {
	jailPath := tmpJailPath()
	cage := jail.New(jailPath, 0755)

	cage.Bind("/bin", "/bin")
	cage.Bind("/lib", "/lib")
	cage.Bind("/usr", "/usr")

	err := cage.Build()
	require.Error(t, err, "Build() is expected to fail in this test")

	_, statErr := os.Stat(cage.Path())
	require.True(t, os.IsNotExist(statErr), "Jail path should not exist")
}

func TestJailBuildCleansUpWhenMountFails(t *testing.T) {
	jailPath := tmpJailPath()
	cage := jail.New(jailPath, 0755)
	cage.Bind("/foo", "/this/path/does/not/exist/so/mount/will/fail")

	err := cage.Build()
	require.Error(t, err, "Build() is expected to fail in this test")

	_, statErr := os.Stat(cage.Path())
	require.True(t, os.IsNotExist(statErr), "Jail path should have been cleaned up")
}

func TestJailDispose(t *testing.T) {
	assert := assert.New(t)

	jailPath := tmpJailPath()
	cage := jail.New(jailPath, 0755)

	err := cage.Build()
	assert.NoError(err)

	err = cage.Dispose()
	assert.NoError(err)

	_, err = os.Stat(cage.Path())
	assert.Error(err, "Jail path should not exist after Jail.Dispose()")
}

func TestJailDisposeDoNotFailOnMissingPath(t *testing.T) {
	assert := assert.New(t)

	jailPath := tmpJailPath()
	cage := jail.New(jailPath, 0755)

	_, err := os.Stat(cage.Path())
	assert.Error(err, "Jail path should not exist")

	err = cage.Dispose()
	assert.NoError(err)
}

func TestJailWithFiles(t *testing.T) {
	tests := []struct {
		name        string
		directories []string
		files       []string
		error       bool
	}{
		{
			name:        "Happy path",
			directories: []string{"/tmp", "/tmp/foo", "/bar"},
		},
		{
			name:        "Missing direcories in path",
			directories: []string{"/tmp/foo/bar"},
			error:       true,
		},
		{
			name:        "copy /etc/resolv.conf",
			directories: []string{"/etc"},
			files:       []string{"/etc/resolv.conf"},
		},
		{
			name:  "copy /etc/resolv.conf without creating /etc",
			files: []string{"/etc/resolv.conf"},
			error: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert := assert.New(t)

			cage := jail.TimestampedJail("jail-mkdir", 0755)
			for _, dir := range test.directories {
				cage.MkDir(dir, 0755)
			}
			for _, file := range test.files {
				if err := cage.Copy(file); err != nil {
					t.Errorf("Can't prepare copy of %s inside the jail. %s", file, err)
				}
			}

			err := cage.Build()
			defer cage.Dispose()

			if test.error {
				assert.Error(err)
			} else {
				assert.NoError(err)

				for _, dir := range test.directories {
					_, err := os.Stat(path.Join(cage.Path(), dir))
					assert.NoError(err, "jailed dir should exist")
				}

				for _, file := range test.files {
					_, err := os.Stat(path.Join(cage.Path(), file))
					assert.NoError(err, "Jailed file should exist")
				}
			}
		})
	}
}

func TestJailCopyTo(t *testing.T) {
	assert := assert.New(t)

	content := "hello"

	cage := jail.TimestampedJail("check-file-copy", 0755)

	tmpFile, err := ioutil.TempFile("", "dummy-file")
	if err != nil {
		t.Error("Can't create temporary file")
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(content)

	filePath := tmpFile.Name()
	jailedFilePath := cage.ExternalPath(path.Base(filePath))

	err = cage.CopyTo(path.Base(filePath), filePath)
	assert.NoError(err)

	err = cage.Build()
	defer cage.Dispose()
	assert.NoError(err)

	jailedFI, err := os.Stat(jailedFilePath)
	assert.NoError(err)

	fi, err := os.Stat(filePath)
	assert.NoError(err)

	assert.Equal(fi.Mode(), jailedFI.Mode(), "jailed file should preserve file mode")
	assert.Equal(fi.Size(), jailedFI.Size(), "jailed file should have same size of original file")

	jailedContent, err := ioutil.ReadFile(jailedFilePath)
	assert.NoError(err)
	assert.Equal(content, string(jailedContent), "jailed file should preserve file content")
}

func TestJailLazyUnbind(t *testing.T) {
	if os.Geteuid() != 0 || runtime.GOOS != "linux" {
		t.Skip("chroot binding requires linux and root permissions")
	}

	assert := assert.New(t)

	toBind, err := ioutil.TempDir("", "to-bind")
	require.NoError(t, err)
	defer os.RemoveAll(toBind)

	tmpFilePath := path.Join(toBind, "a-file")
	tmpFile, err := os.OpenFile(tmpFilePath, os.O_CREATE, 0644)
	require.NoError(t, err)
	tmpFile.Close()

	jailPath := tmpJailPath()
	cage := jail.New(jailPath, 0755)

	cage.MkDir("/my-bind", 0755)
	cage.Bind("/my-bind", toBind)

	err = cage.Build()
	assert.NoError(err, "jail build failed")

	bindedTmpFilePath := cage.ExternalPath("/my-bind/a-file")
	f, err := os.Open(bindedTmpFilePath)
	assert.NoError(err, "temporary file not binded")
	require.NotNil(t, f)

	err = cage.LazyUnbind()
	assert.NoError(err, "lazy unbind failed")

	f.Close()
	_, err = os.Stat(bindedTmpFilePath)
	assert.Error(err, "lazy unbind should remove mount-point after file close")

	err = cage.Dispose()
	assert.NoError(err, "dispose failed")

	_, err = os.Stat(cage.Path())
	assert.Error(err, "Jail path should not exist after Jail.Dispose()")

	_, err = os.Stat(tmpFilePath)
	assert.NoError(err, "disposing a jail should not delete files under binded directories")
}
