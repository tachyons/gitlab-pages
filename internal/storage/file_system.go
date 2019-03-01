package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"

	"gitlab.com/gitlab-org/gitlab-pages/internal/client"
)

type fileSystem struct {
	*client.LookupPath
}

func (f *fileSystem) rootPath() string {
	fullPath, err := filepath.EvalSymlinks(filepath.Join(f.Path))
	if err != nil {
		return ""
	}

	return fullPath
}

func (f *fileSystem) resolvePath(path string) (string, error) {
	fullPath := filepath.Join(f.rootPath(), path)
	fullPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return "", err
	}

	// The requested path resolved to somewhere outside of the root directory
	if !strings.HasPrefix(fullPath, f.rootPath()) {
		return "", fmt.Errorf("%q should be in %q", fullPath, f.rootPath())
	}

	return fullPath, nil
}

func (f *fileSystem) Resolve(path string) (string, error) {
	fullPath, err := f.resolvePath(path)
	if err != nil {
		return "", err
	}

	return fullPath[len(f.rootPath()):], nil
}

func (f *fileSystem) Stat(path string) (os.FileInfo, error) {
	fullPath, err := f.resolvePath(path)
	if err != nil {
		return nil, err
	}

	return os.Lstat(fullPath)
}

func (f *fileSystem) Open(path string) (File, os.FileInfo, error) {
	fullPath, err := f.resolvePath(path)
	if err != nil {
		return nil, nil, err
	}

	file, err := os.OpenFile(fullPath, os.O_RDONLY|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, nil, err
	}

	fileInfo, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, nil, err
	}

	return file, fileInfo, err
}

func (f *fileSystem) Close() {
}
