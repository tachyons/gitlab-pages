package fileresolver

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

var (
	errIsDirectory        = errors.New("location error accessing directory where file expected")
	errNoExtension        = errors.New("error accessing a path without an extension")
	errFileNotFound       = errors.New("file not found")
	errNotRegularFile     = errors.New("not a regular file")
	errFileNotInPublicDir = errors.New("file found outside of public directory")
)

func ResolveFilePath(lookupPath, subPath, requestURLPath string) (string, error) {
	fullPath, err := resolvePath(lookupPath, subPath)
	if err != nil {
		if err == errIsDirectory {
			// try to resolve index.html from the path we're currently in
			if endsWithSlash(requestURLPath) {
				fullPath, err = resolvePath(lookupPath, subPath, "index.html")
				if err != nil {
					return "", err
				}
				return fullPath, nil
			}
		} else if err == errNoExtension {
			return resolvePath(lookupPath, strings.TrimSuffix(subPath, "/")+".html")
		}

		return "", err
	}

	return fullPath, nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func resolvePath(publicPath string, subPath ...string) (string, error) {
	// Ensure that publicPath always ends with "/"
	publicPath = strings.TrimSuffix(publicPath, "/") + "/"

	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := publicPath + strings.Join(subPath, "/")
	fullPath, err := filepath.EvalSymlinks(testPath)
	if err != nil {
		if endsWithoutHTMLExtension(testPath) {
			return "", errNoExtension
		}

		return "", errFileNotFound
	}

	// The requested path resolved to somewhere outside of the public/ directory
	if !strings.HasPrefix(fullPath, publicPath) && fullPath != filepath.Clean(publicPath) {
		return "", errFileNotInPublicDir
	}

	fi, err := os.Lstat(fullPath)
	if err != nil {
		return "", errFileNotFound
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return "", errIsDirectory
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return "", errNotRegularFile
	}

	return fullPath, nil
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

func endsWithoutHTMLExtension(path string) bool {
	return !strings.HasSuffix(path, ".html")
}
