package fileresolver

import (
	"archive/zip"
	"errors"
	"fmt"
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

type evalSymlinkFunc func(string) (string, error)

func ResolveFilePath(lookupPath, subPath, requestURLPath string, evalSymLink evalSymlinkFunc) (string, error) {
	fullPath, err := resolvePath(evalSymLink, lookupPath, subPath)
	if err != nil {
		if err == errIsDirectory {
			fmt.Println("we should come here first")
			// try to resolve index.html from the path we're currently in
			if endsWithSlash(requestURLPath) {
				fmt.Println("aand then here?")
				fullPath, err = resolvePath(evalSymLink, lookupPath, subPath, "index.html")
				if err != nil {
					return "", err
				}
				fmt.Printf("and the ending result: %q\n\n", fullPath)
				return fullPath, nil
			}
		} else if err == errNoExtension {
			return resolvePath(evalSymLink, lookupPath, strings.TrimSuffix(subPath, "/")+".html")
		}

		return "", err
	}

	return fullPath, nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func resolvePath(evalSymLink evalSymlinkFunc, publicPath string, subPath ...string) (string, error) {
	// Ensure that publicPath always ends with "/"
	publicPath = strings.TrimSuffix(publicPath, "/") + "/"

	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)

	testPath := publicPath + strings.Join(subPath, "/")

	fullPath, err := evalSymLink(testPath)
	if err != nil {
		// simpler to return errFileNotFound instead of the other possible errors
		return "", errFileNotFound
	}
	fmt.Printf("publicPath:%q\ntestPath: %q\nfullPath:%q\n\n",
		publicPath, testPath, fullPath)

	for k, s := range subPath {
		fmt.Printf("subpath: %d-%q\n", k, s)
	}
	// if the original testPath ends in with / and the fullPath has no extension, assume it's a directory
	if endsWithSlash(testPath) && endsWithoutHTMLExtension(fullPath) {
		return "", errIsDirectory
	} else if endsWithoutHTMLExtension(fullPath) {
		return "", errNoExtension
	}

	// panic("why")
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

func openZipFile(fullPath string, archive *zip.Reader) (*zip.File, error) {
	return nil, nil
}
func openFSFile(fullPath string) (*os.File, error) {
	fi, err := os.Lstat(fullPath)
	if err != nil {
		return nil, errFileNotFound
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return nil, errIsDirectory
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return nil, errNotRegularFile
	}

	return os.Open(fullPath)
}
