package jail

import (
	"fmt"
	"io"
	"os"
	"path"
	"time"
)

type pathAndMode struct {
	path string
	mode os.FileMode
}

// Jail is a Chroot jail builder
type Jail struct {
	directories []pathAndMode
	files       map[string]pathAndMode
	bindMounts  map[string]string
}

// New returns a Jail for path
func New(path string, perm os.FileMode) *Jail {
	return &Jail{
		directories: []pathAndMode{pathAndMode{path: path, mode: perm}},
		files:       make(map[string]pathAndMode),
		bindMounts:  make(map[string]string),
	}
}

// TimestampedJail return a Jail with Path composed by prefix and current timestamp
func TimestampedJail(prefix string, perm os.FileMode) *Jail {
	jailPath := path.Join(os.TempDir(), fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano()))

	return New(jailPath, perm)
}

// Path returns the path of the jail
func (j *Jail) Path() string {
	return j.directories[0].path
}

// Build creates the jail, making directories and copying files
func (j *Jail) Build() error {
	for _, dir := range j.directories {
		if err := os.Mkdir(dir.path, dir.mode); err != nil {
			return fmt.Errorf("Can't create directory %q. %s", dir.path, err)
		}
	}

	for dest, src := range j.files {
		if err := copyFile(dest, src.path, src.mode); err != nil {
			return fmt.Errorf("Can't copy %q -> %q. %s", src.path, dest, err)
		}
	}

	return j.mount()
}

// Dispose erases everything inside the jail
func (j *Jail) Dispose() error {
	err := j.unmount()
	if err != nil {
		return err
	}

	err = os.RemoveAll(j.Path())
	if err != nil {
		return fmt.Errorf("Can't delete jail %q. %s", j.Path(), err)
	}

	return nil
}

// MkDir enqueue a mkdir operation at jail building time
func (j *Jail) MkDir(path string, perm os.FileMode) {
	j.directories = append(j.directories, pathAndMode{path: j.ExternalPath(path), mode: perm})
}

// CopyTo enqueues a file copy operation at jail building time
func (j *Jail) CopyTo(dest, src string) error {
	fi, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("Can't stat %q. %s", src, err)
	}

	if fi.IsDir() {
		return fmt.Errorf("Can't copy directories. %s", src)
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

// LazyUnbind detaches all binded mountpoints
func (j *Jail) LazyUnbind() error {
	return j.unmount()
}

// ExternalPath converts a jail internal path to the equivalent jail external path
func (j *Jail) ExternalPath(internal string) string {
	return path.Join(j.Path(), internal)
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
