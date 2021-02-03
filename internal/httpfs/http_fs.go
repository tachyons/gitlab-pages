package httpfs

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

var (
	errInvalidChar = errors.New("http: invalid character in file path")
)

// fileSystemPaths implements the http.FileSystem interface
type fileSystemPaths struct {
	allowedPaths []string
}

// NewFileSystemPath creates a new fileSystemPaths that can be used to register
// a file:// protocol with an http.Transport
func NewFileSystemPath(allowedPaths []string) http.FileSystem {
	return &fileSystemPaths{
		allowedPaths: allowedPaths,
	}
}

// Open a file by name if it exists inside the allowedPaths
func (p *fileSystemPaths) Open(name string) (http.File, error) {
	// taken from http.Dir#open https://golang.org/src/net/http/fs.go?s=2108:2152#L70
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errInvalidChar
	}

	absPath, err := filepath.Abs(filepath.FromSlash(path.Clean("/" + name)))
	if err != nil {
		return nil, err
	}

	for _, allowedPath := range p.allowedPaths {
		fmt.Printf("hasPrefix: %t\n", strings.HasPrefix(absPath, allowedPath+"/"))
		if strings.HasPrefix(absPath, allowedPath+"/") {
			return os.Open(absPath)
		}
	}

	// os.ErrPermission is converted to http.StatusForbidden
	// https://github.com/golang/go/blob/release-branch.go1.15/src/net/http/fs.go#L635
	return nil, os.ErrPermission
}
