package storage

import (
	"archive/zip"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/client"
)

const zipDeployPath = "public"
const maxSymlinkSize = 4096
const maxSymlinkDepth = 3

type zipStorage struct {
	*client.LookupPath

	archive *zip.ReadCloser
}

func (z *zipStorage) find(path string) *zip.File {
	// This is O(n) search, very, very, very slow
	for _, file := range z.archive.File {
		if file.Name == path || file.Name == path+"/" {
			return file
		}
	}

	return nil
}

func (z *zipStorage) readSymlink(file *zip.File) (string, error) {
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

func (z *zipStorage) resolveUnchecked(path string) (*zip.File, error) {
	// limit the resolve depth of symlink
	for depth := 0; depth < maxSymlinkDepth; depth++ {
		file := z.find(path)
		if file == nil {
			break
		}

		targetPath, err := z.readSymlink(file)
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

func (z *zipStorage) resolvePublic(path string) (string, *zip.File, error) {
	path = filepath.Join(zipDeployPath, path)
	file, err := z.resolveUnchecked(path)
	if err != nil {
		return "", nil, err
	}

	if !strings.HasPrefix(file.Name, zipDeployPath+"/") {
		return "", nil, fmt.Errorf("%q: is not in %s/", file.Name, zipDeployPath)
	}

	return file.Name[len(zipDeployPath)+1:], file, nil
}

func (z *zipStorage) Resolve(path string) (string, error) {
	targetPath, _, err := z.resolvePublic(path)
	if err != nil {
		println("Resolve", path, "ERROR=", err.Error())
	} else {
		println("Resolve", path, "TARGET_PATH=", targetPath)
	}
	return targetPath, err
}

func (z *zipStorage) Stat(path string) (os.FileInfo, error) {
	_, file, err := z.resolvePublic(path)
	if err != nil {
		println("Stat", path, "ERROR=", err.Error())
		return nil, err
	}

	println("Stat", path, "FILE=", file.Name, file.FileInfo())
	return file.FileInfo(), nil
}

func (z *zipStorage) Open(path string) (File, os.FileInfo, error) {
	_, file, err := z.resolvePublic(path)
	if err != nil {
		println("Open", path, "ERROR=", err.Error())
		return nil, nil, err
	}

	rc, err := file.Open()
	if err != nil {
		println("Open", path, "ERROR=", err.Error())
		return nil, nil, err
	}

	println("Open", path, "FILE=", file.Name, file.FileInfo())
	return rc, file.FileInfo(), nil
}

func (z *zipStorage) Close() {
	println("Close")
	z.archive.Close()
}

func newZipStorage(lookupPath *client.LookupPath) (S, error) {
	archive, err := zip.OpenReader(lookupPath.ArchivePath)
	if err != nil {
		return nil, err
	}

	return &zipStorage{LookupPath: lookupPath, archive: archive}, nil
}
