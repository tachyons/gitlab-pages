package jail

import (
	"fmt"
	"io"
	"os"
	"path"
	"syscall"
	"time"

	"golang.org/x/sys/unix"
)

type pathAndMode struct {
	path string
	mode os.FileMode

	// Only respected if mode is os.ModeCharDevice
	rdev int
}

// Jail is a Chroot jail builder
type Jail struct {
	root        string
	deleteRoot  bool
	directories []pathAndMode
	files       map[string]pathAndMode
	bindMounts  map[string]string
}

// Into returns a Jail on path, assuming it already exists on disk. On disposal,
// the jail *will not* remove the path
func Into(path string) *Jail {
	return &Jail{
		root:       path,
		deleteRoot: false,
		files:      make(map[string]pathAndMode),
		bindMounts: make(map[string]string),
	}
}

// Create returns a Jail on path, creating the directory if needed. On disposal,
// the jail will remove the path
func Create(path string, perm os.FileMode) *Jail {
	jail := Into(path)
	jail.deleteRoot = true
	jail.directories = append(jail.directories, pathAndMode{path: path, mode: perm})

	return jail
}

// CreateTimestamped returns a Jail on a path composed by prefix and current
// timestamp, creating the directory. On disposal, the jail will remove the path
func CreateTimestamped(prefix string, perm os.FileMode) *Jail {
	jailPath := path.Join(os.TempDir(), fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()))

	return Create(jailPath, perm)
}

// Path returns the path of the jail
func (j *Jail) Path() string {
	return j.root
}

// Build creates the jail, making directories and copying files. If an error
// setting up is encountered, a best-effort attempt will be made to remove any
// partial state before returning the error
func (j *Jail) Build() error {
	// Simplify error-handling in this method. It's unsafe to run os.RemoveAll()
	// across a bind mount. Only one is needed at present, and this restriction
	// means there's no need to handle the case where one of several mounts
	// failed in j.mount()
	//
	// Make j.mount() robust before removing this restriction, at the risk of
	// extreme data loss
	if len(j.bindMounts) > 1 {
		return fmt.Errorf("BUG: jail does not currently support multiple bind mounts")
	}

	for _, dir := range j.directories {
		if err := os.MkdirAll(dir.path, dir.mode); err != nil {
			j.removeAll()
			return fmt.Errorf("can't create directory %q. %s", dir.path, err)
		}
	}

	for dest, src := range j.files {
		if err := handleFile(dest, src); err != nil {
			j.removeAll()
			return fmt.Errorf("can't copy %q -> %q. %s", src.path, dest, err)
		}
	}

	if err := j.mount(); err != nil {
		// Only one bind mount is supported. If it failed to mount, there is
		// nothing to unmount, so it is safe to run removeAll() here.
		j.removeAll()
		return err
	}

	return nil
}

func (j *Jail) removeAll() error {
	// Deleting the root will remove all child directories, so there's no need
	// to traverse files and directories
	if j.deleteRoot {
		if err := os.RemoveAll(j.Path()); err != nil {
			return fmt.Errorf("can't delete jail %q. %s", j.Path(), err)
		}
	} else {
		for path := range j.files {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("can't delete file in jail %q: %s", path, err)
			}
		}

		// Iterate directories in reverse to remove children before parents
		for i := len(j.directories) - 1; i >= 0; i-- {
			dest := j.directories[i]
			if err := os.Remove(dest.path); err != nil {
				return fmt.Errorf("can't delete directory in jail %q: %s", dest.path, err)
			}
		}
	}

	return nil
}

// Dispose erases everything inside the jail
func (j *Jail) Dispose() error {
	if err := j.unmount(); err != nil {
		return err
	}

	if err := j.removeAll(); err != nil {
		return fmt.Errorf("can't delete jail %q. %s", j.Path(), err)
	}

	return nil
}

// MkDir enqueue a mkdir operation at jail building time
func (j *Jail) MkDir(path string, perm os.FileMode) {
	j.directories = append(j.directories, pathAndMode{path: j.ExternalPath(path), mode: perm})
}

// CharDev enqueues an mknod operation for the given character device at jail
// building time
func (j *Jail) CharDev(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("can't stat %q: %s", path, err)
	}

	if (fi.Mode() & os.ModeCharDevice) == 0 {
		return fmt.Errorf("can't mknod %q: not a character device", path)
	}

	// Read the device number from the underlying unix implementation of stat()
	sys, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("couldn't determine rdev for %q", path)
	}

	jailedDest := j.ExternalPath(path)
	j.files[jailedDest] = pathAndMode{
		path: path,
		mode: fi.Mode(),
		rdev: int(sys.Rdev),
	}

	return nil
}

// CopyTo enqueues a file copy operation at jail building time
func (j *Jail) CopyTo(dest, src string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("can't stat %q. %s", src, err)
	}

	if fi.IsDir() {
		return fmt.Errorf("can't copy directories. %s", src)
	}

	jailedDest := j.ExternalPath(dest)
	j.files[jailedDest] = pathAndMode{
		path: src,
		mode: fi.Mode(),
	}

	return nil
}

// Copy enqueues a file copy operation at jail building time
func (j *Jail) Copy(path string) error {
	return j.CopyTo(path, path)
}

// Bind enqueues a bind mount operation at jail building time
func (j *Jail) Bind(dest, src string) {
	jailedDest := j.ExternalPath(dest)
	j.bindMounts[jailedDest] = src
}

// ExternalPath converts a jail internal path to the equivalent jail external path
func (j *Jail) ExternalPath(internal string) string {
	return path.Join(j.Path(), internal)
}

func handleFile(dest string, src pathAndMode) error {
	// Using `io.Copy` on a character device simply doesn't work
	if (src.mode & os.ModeCharDevice) > 0 {
		return createCharacterDevice(dest, src)
	}

	// FIXME: currently, symlinks, block devices, named pipes and other
	// non-regular files will be `Open`ed and have that content streamed to a
	// regular file inside the chroot. This is actually desired behaviour for,
	// e.g., `/etc/resolv.conf`, but was very surprising
	return copyFile(dest, src.path, src.mode)
}

func createCharacterDevice(dest string, src pathAndMode) error {
	unixMode := uint32(src.mode.Perm() | syscall.S_IFCHR)

	return unix.Mknod(dest, unixMode, src.rdev)
}

func copyFile(dest, src string, perm os.FileMode) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	destFile, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	return err
}
