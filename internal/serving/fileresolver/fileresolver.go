package fileresolver

import (
	"errors"
	"fmt"
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

type evalSymlinkFunc func(string) (string, error)

// ResolveFilePath takes a archivePath and any subPath to determine the file location.
// Requires the original requestURLPath to try to resolve index.html
// Requires an evalSymlinkFunc to determine if the file exists or not. Useful for resolving files in disk
func ResolveFilePath(lookupPath, subPath, urlPath string, evalSymLink evalSymlinkFunc) (string, error) {
	fmt.Printf("THIS IS IT!:\nlookup.path: %q\nsubPath:%q\nurlPath:%q\n", lookupPath, subPath, urlPath)

	fullPath, err := resolvePath(evalSymLink, lookupPath, subPath)
	if err != nil {
		if err == errIsDirectory {
			// try to resolve index.html from the path we're currently in
			if endsWithSlash(urlPath) {
				return resolvePath(evalSymLink, lookupPath, subPath, "index.html")
			}
			return "", err
		} else if err == errNoExtension {
			// assume .html extension and try to resolve
			return resolvePath(evalSymLink, lookupPath, strings.TrimSuffix(subPath, "/")+".html")
		}

		return "", err
	}

	return fullPath, nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
// Takes a `evalSymLinkFunc` to try to follow any symlinks. For disk use `filepath.EvalSymlinks`.
// Returns the resolved fullPath, fileName (filepath.Base) and error
func resolvePath(evalSymLink evalSymlinkFunc, publicPath string, subPath ...string) (string, error) {
	// Ensure that publicPath always ends with "/"
	publicPath = strings.TrimSuffix(publicPath, "/") + "/"

	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := publicPath + strings.Join(cleanEmpty(subPath), "/")
	if endsWithSlash(testPath) {
		return "", errIsDirectory
	}

	fullPath, err := evalSymLink(testPath)
	if err != nil {
		// if file doesn't exist and ends without extension
		if endsWithoutHTMLExtension(testPath) {
			return "", errNoExtension
		}

		return "", errFileNotFound
	}

	// The requested path resolved to somewhere outside of the public/ directory
	if !strings.HasPrefix(fullPath, publicPath) && fullPath != filepath.Clean(publicPath) {
		return "", errFileNotInPublicDir
	}

	return fullPath, nil
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

func endsWithoutHTMLExtension(path string) bool {
	return !strings.HasSuffix(path, ".html")
}

// cleanEmpty removes empty string elements in the slice
func cleanEmpty(in []string) []string {
	var out []string

	for _, x := range in {
		if x != "" {
			out = append(out, x)
		}
	}

	return out
}
