// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on https://golang.org/src/net/http/fs.go

package httpfs

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"gitlab.com/gitlab-org/labkit/log"
)

var (
	errInvalidChar = errors.New("http: invalid character in file path")
)

// fileSystemPaths implements the http.FileSystem interface
type fileSystemPaths struct {
	allowedPaths []string
	// workaround for https://gitlab.com/gitlab-org/gitlab/-/issues/326117#note_546346101
	// where daemon-inplace-chroot=true fails to serve zip archives when
	// pages_serve_with_zip_file_protocol is enabled
	// TODO: evaluate if we need to remove this field when we remove
	// chroot https://gitlab.com/gitlab-org/gitlab-pages/-/issues/561
	chrootPath string
}

// NewFileSystemPath creates a new fileSystemPaths that can be used to register
// a file:// protocol with an http.Transport.
// When the daemon runs inside a chroot we need to strip chrootPath out of each
// of the allowedPaths so that we are able to find the file correctly inside
// the chroot. When Open is called, the same chrootPath will be stripped out of
// the full filepath.
func NewFileSystemPath(allowedPaths []string, chrootPath string) (http.FileSystem, error) {
	for k, allowedPath := range allowedPaths {
		strippedPath, err := stripChrootPath(ensureEndingSlash(allowedPath), chrootPath)
		if err != nil {
			return nil, err
		}

		allowedPaths[k], err = filepath.Abs(strippedPath)
		if err != nil {
			return nil, err
		}
	}

	return &fileSystemPaths{
		allowedPaths: allowedPaths,
		chrootPath:   chrootPath,
	}, nil
}

// Open a file by name if it exists inside the allowedPaths
func (p *fileSystemPaths) Open(name string) (http.File, error) {
	// taken from http.Dir#open https://golang.org/src/net/http/fs.go?s=2108:2152#L70
	if filepath.Separator != '/' && strings.ContainsRune(name, filepath.Separator) {
		return nil, errInvalidChar
	}

	cleanedPath := filepath.FromSlash(path.Clean("/" + name))

	// since deamon can run in a chroot, we allow to define a chroot path that will be stripped from
	// the FS location
	// TODO: evaluate if we need to remove this check when we remove chroot
	// https://gitlab.com/gitlab-org/gitlab-pages/-/issues/561
	strippedPath, err := stripChrootPath(cleanedPath, p.chrootPath)
	if err != nil {
		log.WithError(err).Error(os.ErrPermission)

		return nil, os.ErrPermission
	}

	absPath, err := filepath.Abs(strippedPath)
	if err != nil {
		return nil, err
	}
	for _, allowedPath := range p.allowedPaths {
		// allowedPath may be a single / in chroot so we need to ensure it's not double slash
		if strings.HasPrefix(absPath, ensureEndingSlash(allowedPath)) {
			return os.Open(absPath)
		}
	}

	log.WithError(os.ErrPermission).Errorf("requested filepath %q not in allowed paths: %q",
		absPath, strings.Join(p.allowedPaths, string(os.PathListSeparator)))

	// os.ErrPermission is converted to http.StatusForbidden
	// https://github.com/golang/go/blob/release-branch.go1.15/src/net/http/fs.go#L635
	return nil, os.ErrPermission
}

func ensureEndingSlash(path string) string {
	if strings.HasSuffix(path, "/") {
		return path
	}

	return path + "/"
}

func stripChrootPath(path, chrootPath string) (string, error) {
	if chrootPath == "" {
		return path, nil
	}

	if !strings.HasPrefix(path, chrootPath+"/") {
		return "", fmt.Errorf("allowed path %q is not in chroot path %q", path, chrootPath)
	}

	// path will contain a leading `/`
	path = path[len(chrootPath):]

	return path, nil
}
