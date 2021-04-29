package jail_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/jail"
)

func tmpJailPath() string {
	return path.Join(os.TempDir(), fmt.Sprintf("my-jail-%d", time.Now().Unix()))
}

func TestTimestampedJails(t *testing.T) {
	require := require.New(t)

	prefix := "jail"
	var mode os.FileMode = 0755

	j1 := jail.CreateTimestamped(prefix, mode)
	j2 := jail.CreateTimestamped(prefix, mode)

	require.NotEqual(j1.Path(), j2.Path())
}

func TestJailPath(t *testing.T) {
	require := require.New(t)

	jailPath := tmpJailPath()
	cage := jail.Create(jailPath, 0755)

	require.Equal(jailPath, cage.Path())
}

func TestJailBuild(t *testing.T) {
	require := require.New(t)

	jailPath := tmpJailPath()
	cage := jail.Create(jailPath, 0755)

	_, err := os.Stat(cage.Path())
	require.Error(err, "Jail path should not exist before Jail.Build()")

	err = cage.Build()
	require.NoError(err)
	defer cage.Dispose()

	_, err = os.Stat(cage.Path())
	require.NoError(err, "Jail path should exist after Jail.Build()")
}

func TestJailOnlySupportsOneBindMount(t *testing.T) {
	jailPath := tmpJailPath()
	cage := jail.Create(jailPath, 0755)

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
	cage := jail.Create(jailPath, 0755)
	cage.Bind("/foo", "/this/path/does/not/exist/so/mount/will/fail")

	err := cage.Build()
	require.Error(t, err, "Build() is expected to fail in this test")

	_, statErr := os.Stat(cage.Path())
	require.True(t, os.IsNotExist(statErr), "Jail path should have been cleaned up")
}

func TestJailDispose(t *testing.T) {
	require := require.New(t)

	jailPath := tmpJailPath()
	cage := jail.Create(jailPath, 0755)

	err := cage.Build()
	require.NoError(err)

	err = cage.Dispose()
	require.NoError(err)

	_, err = os.Stat(cage.Path())
	require.Error(err, "Jail path should not exist after Jail.Dispose()")
}

func TestJailDisposeDoNotFailOnMissingPath(t *testing.T) {
	require := require.New(t)

	jailPath := tmpJailPath()
	cage := jail.Create(jailPath, 0755)

	_, err := os.Stat(cage.Path())
	require.Error(err, "Jail path should not exist")

	err = cage.Dispose()
	require.NoError(err)
}

func TestJailWithCharacterDevice(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Log("This test only works if run as root")
		t.SkipNow()
	}

	// Determine the expected rdev
	fi, err := os.Stat("/dev/urandom")
	require.NoError(t, err)
	sys, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		t.Log("Couldn't determine expected rdev for /dev/urandom, skipping")
		t.SkipNow()
	}

	expectedRdev := sys.Rdev

	jailPath := tmpJailPath()
	cage := jail.Create(jailPath, 0755)
	cage.MkDir("/dev", 0755)

	require.NoError(t, cage.CharDev("/dev/urandom"))
	require.NoError(t, cage.Build())
	defer cage.Dispose()

	fi, err = os.Lstat(path.Join(cage.Path(), "/dev/urandom"))
	require.NoError(t, err)

	isCharDev := fi.Mode()&os.ModeCharDevice == os.ModeCharDevice
	require.True(t, isCharDev, "Created file was not a character device")

	sys, ok = fi.Sys().(*syscall.Stat_t)
	require.True(t, ok, "Couldn't determine rdev of created character device")
	require.Equal(t, expectedRdev, sys.Rdev, "Incorrect rdev for /dev/urandom")
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
			require := require.New(t)

			cage := jail.CreateTimestamped("jail-mkdir", 0755)
			for _, dir := range test.directories {
				cage.MkDir(dir, 0755)
			}
			for _, file := range test.files {
				if err := cage.Copy(file); err != nil {
					t.Errorf("can't prepare copy of %s inside the jail. %s", file, err)
				}
			}

			err := cage.Build()
			defer cage.Dispose()

			if test.error {
				require.Error(err)
			} else {
				require.NoError(err)

				for _, dir := range test.directories {
					_, err := os.Stat(path.Join(cage.Path(), dir))
					require.NoError(err, "jailed dir should exist")
				}

				for _, file := range test.files {
					_, err := os.Stat(path.Join(cage.Path(), file))
					require.NoError(err, "Jailed file should exist")
				}
			}
		})
	}
}

func TestJailCopyTo(t *testing.T) {
	require := require.New(t)

	content := "hello"

	cage := jail.CreateTimestamped("check-file-copy", 0755)

	tmpFile, err := ioutil.TempFile("", "dummy-file")
	if err != nil {
		t.Error("Can't create temporary file")
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString(content)

	filePath := tmpFile.Name()
	jailedFilePath := cage.ExternalPath(path.Base(filePath))

	err = cage.CopyTo(path.Base(filePath), filePath)
	require.NoError(err)

	err = cage.Build()
	defer cage.Dispose()
	require.NoError(err)

	jailedFI, err := os.Stat(jailedFilePath)
	require.NoError(err)

	fi, err := os.Stat(filePath)
	require.NoError(err)

	require.Equal(fi.Mode(), jailedFI.Mode(), "jailed file should preserve file mode")
	require.Equal(fi.Size(), jailedFI.Size(), "jailed file should have same size of original file")

	jailedContent, err := ioutil.ReadFile(jailedFilePath)
	require.NoError(err)
	require.Equal(content, string(jailedContent), "jailed file should preserve file content")
}

func TestJailIntoOnlyCleansSubpaths(t *testing.T) {
	jailPath := tmpJailPath()
	require.NoError(t, os.MkdirAll(jailPath, 0755))
	defer os.RemoveAll(jailPath)

	chroot := jail.Into(jailPath)
	chroot.MkDir("/etc", 0755)
	chroot.Copy("/etc/resolv.conf")
	require.NoError(t, chroot.Build())
	require.NoError(t, chroot.Dispose())

	_, err := os.Stat(path.Join(jailPath, "/etc/resolv.conf"))
	require.True(t, os.IsNotExist(err), "/etc/resolv.conf in jail was not removed")
	_, err = os.Stat(path.Join(jailPath, "/etc"))
	require.True(t, os.IsNotExist(err), "/etc in jail was not removed")
	_, err = os.Stat(jailPath)
	require.NoError(t, err, "/ in jail (corresponding to external directory) was removed")
}

func TestJailIntoCleansNestedDirs(t *testing.T) {
	jailPath := tmpJailPath()
	require.NoError(t, os.MkdirAll(jailPath, 0755))
	defer os.RemoveAll(jailPath)

	chroot := jail.Into(jailPath)

	// These need to be cleaned up in reverse order
	chroot.MkDir("/way", 0755)
	chroot.MkDir("/way/down", 0755)
	chroot.MkDir("/way/down/here", 0755)

	require.NoError(t, chroot.Build())
	require.NoError(t, chroot.Dispose())

	verify := func(inPath string) {
		_, err := os.Stat(path.Join(jailPath, inPath))
		require.True(t, os.IsNotExist(err), "{} in jail was not removed", inPath)
	}

	verify("/way")
	verify("/way/down")
	verify("/way/down/here")

	_, err := os.Stat(jailPath)
	require.NoError(t, err, "/ in jail (corresponding to external directory) was removed")
}

func TestJailIntoMkDirFails(t *testing.T) {
	jailPath := tmpJailPath()
	require.NoError(t, os.MkdirAll(jailPath, 0755))
	defer os.RemoveAll(jailPath)

	pagesRoot := "/pages/sub/path"

	chroot := jail.Into(jailPath)
	chroot.MkDir(pagesRoot, 0755)

	err := chroot.Build()

	require.Error(t, err, "err")
	require.Contains(t, err.Error(), "no such file or directory")

	_, err = os.Stat(path.Join(jailPath, pagesRoot))
	require.True(t, os.IsNotExist(err), "%s in jail was not removed", pagesRoot)
}

func TestJailIntoMkDirAll(t *testing.T) {
	jailPath := tmpJailPath()
	require.NoError(t, os.MkdirAll(jailPath, 0755))
	defer os.RemoveAll(jailPath)

	chroot := jail.Into(jailPath)

	chroot.MkDirAll("/way/down/here", 0755)

	require.NoError(t, chroot.Build())
	require.NoError(t, chroot.Dispose())

	verify := func(inPath string) {
		_, err := os.Stat(path.Join(jailPath, inPath))
		require.True(t, os.IsNotExist(err), "{} in jail was not removed", inPath)
	}

	verify("/way")
	verify("/way/down")
	verify("/way/down/here")

	_, err := os.Stat(jailPath)
	require.NoError(t, err, "/ in jail (corresponding to external directory) was not removed")
}
