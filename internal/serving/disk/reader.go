package disk

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Reader struct {
	Group string
}

func (reader *Reader) tryFile(w http.ResponseWriter, r *http.Request, projectName string, subPath ...string) error {
	fullPath, err := reader.resolvePath(projectName, subPath...)

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(r.URL.Path) {
			fullPath, err = reader.resolvePath(projectName, filepath.Join(subPath...), "index.html")
		} else {
			// Concat Host with URL.Path
			redirectPath := "//" + r.Host + "/"
			redirectPath += strings.TrimPrefix(r.URL.Path, "/")

			// Ensure that there's always "/" at end
			redirectPath = strings.TrimSuffix(redirectPath, "/") + "/"
			http.Redirect(w, r, redirectPath, 302)
			return nil
		}
	}

	if locationError, _ := err.(*locationFileNoExtensionError); locationError != nil {
		fullPath, err = reader.resolvePath(projectName, strings.TrimSuffix(filepath.Join(subPath...), "/")+".html")
	}

	if err != nil {
		return err
	}

	return reader.serveFile(w, r, fullPath)
}

func (reader *Reader) tryNotFound(w http.ResponseWriter, r *http.Request, projectName string) error {
	page404, err := reader.resolvePath(projectName, "404.html")
	if err != nil {
		return err
	}

	err = reader.serveCustomFile(w, r, http.StatusNotFound, page404)
	if err != nil {
		return err
	}
	return nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (reader *Reader) resolvePath(projectName string, subPath ...string) (string, error) {
	publicPath := filepath.Join(reader.Group, projectName, "public")

	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := publicPath + "/" + strings.Join(subPath, "/")
	fullPath, err := filepath.EvalSymlinks(testPath)
	if err != nil {
		if endsWithoutHTMLExtension(testPath) {
			return "", &locationFileNoExtensionError{
				FullPath: fullPath,
			}
		}

		return "", err
	}

	// The requested path resolved to somewhere outside of the public/ directory
	if !strings.HasPrefix(fullPath, publicPath+"/") && fullPath != publicPath {
		return "", fmt.Errorf("%q should be in %q", fullPath, publicPath)
	}

	fi, err := os.Lstat(fullPath)
	if err != nil {
		return "", err
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return "", &locationDirectoryError{
			FullPath:     fullPath,
			RelativePath: strings.TrimPrefix(fullPath, publicPath),
		}
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s: is not a regular file", fullPath)
	}

	return fullPath, nil
}

func (reader *Reader) serveFile(w http.ResponseWriter, r *http.Request, origPath string) error {
	fullPath := handleGZip(w, r, origPath)

	file, err := openNoFollow(fullPath)
	if err != nil {
		return err
	}

	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	contentType, err := detectContentType(origPath)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, origPath, fi.ModTime(), file)

	return nil
}

func (reader *Reader) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, origPath string) error {
	fullPath := handleGZip(w, r, origPath)

	// Open and serve content of file
	file, err := openNoFollow(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	contentType, err := detectContentType(origPath)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(code)

	if r.Method != "HEAD" {
		_, err := io.CopyN(w, file, fi.Size())
		return err
	}

	return nil
}
