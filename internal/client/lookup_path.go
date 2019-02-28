package client

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sys/unix"
)

// TODO: This is hack to pass the location
var RootPath string

type LookupPath struct {
	Prefix string `json:"prefix"`
	Path   string `json:"path"`

	NamespaceProject bool   `json:"namespace_project"`
	HTTPSOnly        bool   `json:"https_only"`
	AccessControl    bool   `json:"access_control"`
	ProjectID        uint64 `json:"id"`
}

func (lp *LookupPath) Tail(path string) string {
	if strings.HasPrefix(path, lp.Prefix) {
		return path[len(lp.Prefix):]
	}

	return ""
}

func (lp *LookupPath) rootPath() string {
	fullPath, err := filepath.EvalSymlinks(filepath.Join(RootPath, lp.Path))
	if err != nil {
		return ""
	}

	return fullPath
}

func (lp *LookupPath) resolvePath(path string) (string, error) {
	fullPath := filepath.Join(lp.rootPath(), path)
	fullPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return "", err
	}

	// The requested path resolved to somewhere outside of the public/ directory
	if !strings.HasPrefix(fullPath, lp.rootPath()) {
		return "", fmt.Errorf("%q should be in %q", fullPath, lp.rootPath())
	}

	return fullPath, nil
}

func (lp *LookupPath) Resolve(path string) (string, error) {
	fullPath, err := lp.resolvePath(path)
	if err != nil {
		println("LookupPath::Resolve", lp.rootPath(), "PATH=", path, "FULLPATH=", fullPath, "ERROR", err.Error())
		return "", err
	}

	println("LookupPath::Resolve", lp.rootPath(), "PATH=", path, "FULLPATH=", fullPath, err)
	return fullPath[len(lp.rootPath()):], nil
}

func (lp *LookupPath) Stat(path string) (os.FileInfo, error) {
	fullPath, err := lp.resolvePath(path)
	if err != nil {
		println("LookupPath::Stat", lp.rootPath(), "PATH=", path, "FULLPATH=", fullPath, "ERROR", err.Error())
		return nil, err
	}

	println("LookupPath::Stat", lp.rootPath(), "PATH=", path, "FULLPATH=", fullPath, err)
	return os.Lstat(fullPath)
}

func (lp *LookupPath) Open(path string) (*os.File, error) {
	fullPath, err := lp.resolvePath(path)
	if err != nil {
		println("LookupPath::Open", lp.rootPath(), "PATH=", path, "FULLPATH=", fullPath, "ERROR", err.Error())
		return nil, err
	}

	println("LookupPath::Open", lp.rootPath(), "PATH=", path, "FULLPATH=", fullPath, err)
	return os.OpenFile(fullPath, os.O_RDONLY|unix.O_NOFOLLOW, 0)
}
