package zip

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

const zipDeployPath = "public"
const maxSymlinkSize = 4096
const maxSymlinkDepth = 3

// Reader ..
type Reader struct {
	archive *zip.Reader
}

// New creates a new zip Reader. A reader can find a file inside an archive
// with a depth < `maxSymlinkDepth`
func New(readerAt io.ReaderAt, size int64) (*Reader, error) {
	archive, err := zip.NewReader(readerAt, size)
	if err != nil {
		return nil, err
	}

	return &Reader{archive: archive}, nil
}

func (r *Reader) find(path string) *zip.File {
	// This is O(n) search, very, very, very slow
	for _, file := range r.archive.File {
		if file.Name == path || file.Name == path+"/" {
			return file
		}
	}

	return nil
}

func (r *Reader) readSymlink(file *zip.File) (string, error) {
	fi := file.FileInfo()

	if (fi.Mode() & os.ModeSymlink) != os.ModeSymlink {
		return "", nil
	}

	if fi.Size() > maxSymlinkSize {
		return "", errors.New("symlink size too long")
	}

	rc, err := file.Open()
	if err != nil {
		return "", err
	}
	defer rc.Close()

	data, err := ioutil.ReadAll(rc)
	if err != nil {
		return "", err
	}

	// resolve symlink location relative to current file
	targetPath, err := filepath.Rel(filepath.Dir(file.Name), string(data))
	if err != nil {
		return "", err
	}

	return targetPath, nil
}

func (r *Reader) resolveUnchecked(path string) (*zip.File, error) {
	// limit the resolve depth of symlink
	for depth := 0; depth < maxSymlinkDepth; depth++ {
		file := r.find(path)
		if file == nil {
			break
		}

		targetPath, err := r.readSymlink(file)
		if err != nil {
			return nil, err
		}

		// not a symlink
		if targetPath == "" {
			return file, nil
		}

		path = targetPath
	}

	return nil, fmt.Errorf("%q: not found", path)
}

func (r *Reader) resolvePublic(path string) (string, *zip.File, error) {
	path = filepath.Join(zipDeployPath, path)
	file, err := r.resolveUnchecked(path)
	if err != nil {
		return "", nil, err
	}

	if !strings.HasPrefix(file.Name, zipDeployPath+"/") {
		return "", nil, fmt.Errorf("%q: is not in %s/", file.Name, zipDeployPath)
	}

	return file.Name[len(zipDeployPath)+1:], file, nil
}

// Open returns a ReadCloser to the caller which is responsible of closing. os.FileInfo is also returned in one go.
func (r *Reader) Open(path string) (io.ReadCloser, os.FileInfo, error) {
	_, file, err := r.resolvePublic(path)
	if err != nil {
		return nil, nil, err
	}

	rc, err := file.Open()
	if err != nil {
		return nil, nil, err
	}

	return rc, file.FileInfo(), nil
}
