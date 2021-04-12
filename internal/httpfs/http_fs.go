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
	// TODO: this is a temporary workaround for https://gitlab.com/gitlab-org/gitlab/-/issues/326117#note_546346101
	// where daemon-inplace-chroot=true fails to serve zip archives when pages_serve_with_zip_file_protocol is enabled
	// To be removed after we roll-out zip architecture completely https://gitlab.com/gitlab-org/gitlab-pages/-/issues/561
	chrootPath string
}

// NewFileSystemPath creates a new fileSystemPaths that can be used to register
// a file:// protocol with an http.Transport
func NewFileSystemPath(allowedPaths []string, chrootPath string) (http.FileSystem, error) {
	for k, path := range allowedPaths {
		var err error

		if chrootPath != "" {
			if !strings.HasPrefix(path, chrootPath) {
				return nil, fmt.Errorf("allowed path %q is not in chroot path %q", path, chrootPath)
			}

			// strip chrootPath from each allowedPath
			path = path[len(chrootPath):]
		}

		allowedPaths[k], err = filepath.Abs(path)
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

	absPath := filepath.FromSlash(path.Clean("/" + name))

	// since deamon can run in a chroot, we allow to define a chroot path that will be stripped from
	// the FS location
	// TODO: To be removed after we roll-out zip architecture completely https://gitlab.com/gitlab-org/gitlab-pages/-/issues/561
	if p.chrootPath != "" {
		if !strings.HasPrefix(absPath, p.chrootPath) {
			log.WithError(os.ErrPermission).Errorf(
				"requested filepath %q not in chroot path: %q", absPath, p.chrootPath)

			return nil, os.ErrPermission
		}

		// since we run in a chroot path, strip it
		absPath = absPath[len(p.chrootPath):]
	}

	absPath, err := filepath.Abs(absPath)
	if err != nil {
		return nil, err
	}
	for _, allowedPath := range p.allowedPaths {
		// TODO: To be removed after we roll-out zip architecture completely https://gitlab.com/gitlab-org/gitlab-pages/-/issues/561
		// return early if in chroot and we have stripped p.chrootPath from absPath && allowedPath
		if p.chrootPath != "" && allowedPath == "/" {
			return os.Open(absPath)
		}

		if strings.HasPrefix(absPath, allowedPath+"/") {
			return os.Open(absPath)
		}
	}

	log.WithError(os.ErrPermission).Errorf("requested filepath %q not in allowed paths: %q",
		absPath, strings.Join(p.allowedPaths, string(os.PathListSeparator)))

	// os.ErrPermission is converted to http.StatusForbidden
	// https://github.com/golang/go/blob/release-branch.go1.15/src/net/http/fs.go#L635
	return nil, os.ErrPermission
}
