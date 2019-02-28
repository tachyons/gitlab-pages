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

func (lp *LookupPath) resolvePath(path string) (string, error) {
	fullPath := filepath.Join(lp.Path, path)
	fullPath, err := filepath.EvalSymlinks(fullPath)
	if err != nil {
		return "", err
	}

	// The requested path resolved to somewhere outside of the public/ directory
	if !strings.HasPrefix(fullPath, lp.Path) {
		return "", fmt.Errorf("%q should be in %q", fullPath, lp.Path)
	}

	return fullPath, nil
}

func (lp *LookupPath) Resolve(path string) (string, error) {
	fullPath, err := lp.resolvePath(path)
	println("LookupPath::Resolve", lp.Path, path, fullPath, err)
	if err != nil {
		return "", err
	}

	return fullPath[len(lp.Path):], nil
}

func (lp *LookupPath) Stat(path string) (os.FileInfo, error) {
	fullPath, err := lp.resolvePath(path)
	println("LookupPath::Stat", lp.Path, path, fullPath, err)
	if err != nil {
		return nil, err
	}

	return os.Lstat(fullPath)
}

func (lp *LookupPath) Open(path string) (*os.File, error) {
	fullPath, err := lp.resolvePath(path)
	println("LookupPath::Open", lp.Path, path, fullPath, err)
	if err != nil {
		return nil, err
	}

	return os.OpenFile(fullPath, os.O_RDONLY|unix.O_NOFOLLOW, 0)
}
