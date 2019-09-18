package source

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httputil"
	"golang.org/x/sys/unix"
)

// File struct represent the on-disk source for a GitLab Pages Domain.
type Disk struct {
	Project      string
	Group        string
	CustomDomain bool
}

type locationDirectoryError struct {
	FullPath     string
	RelativePath string
}

type locationFileNoExtensionError struct {
	FullPath string
}

func (l *locationDirectoryError) Error() string {
	return "location error accessing directory where file expected"
}

func (l *locationFileNoExtensionError) Error() string {
	return "error accessing a path without an extension"
}

// Writes to the http.ResponseWritter. Returns true if something was served, false if not.
func (d *Disk) ServeFileHTTP(serving *Serving) bool {
	if d == nil { // TODO do we need that
		httperrors.Serve404(serving.Writer)
		return true
	}

	if d.CustomDomain {
		return d.serveFileFromConfig(serving)
	}

	return d.serveFileFromGroup(serving)
}

// Writes to the http.ResponseWriter. Serves the not found page from a project.
func (d *Disk) ServeNotFoundHTTP(serving *Serving) {
	if d == nil { // TODO do we need that
		httperrors.Serve404(serving.Writer)
		return
	}

	if d.CustomDomain {
		d.serveNotFoundFromConfig(serving)
	} else {
		d.serveNotFoundFromGroup(serving)
	}
}

func (d *Disk) serveFileFromGroup(serving *Serving) bool {
	if !serving.IsProjectFound() {
		httperrors.Serve404(serving.Writer)
		return true
	}

	if d.tryFile(serving.Writer, serving.Request, serving.Project, serving.SubPath) == nil {
		return true
	}

	return false
}

func (d *Disk) serveNotFoundFromGroup(serving *Serving) {
	if !serving.IsProjectFound() {
		httperrors.Serve404(serving.Writer)
		return
	}

	// Try serving custom not-found page
	if d.tryNotFound(serving.Writer, serving.Request, serving.Project) == nil {
		return
	}

	// Generic 404
	httperrors.Serve404(serving.Writer)
}

// TODO can we use r.project instead of d.project?
func (d *Disk) serveFileFromConfig(serving *Serving) bool {
	// Try to serve file for http://host/... => /group/project/...
	if d.tryFile(serving.Writer, serving.Request, serving.Project, serving.requestPath()) == nil {
		return true
	}

	return false
}

// TODO can we use r.project instead of d.project?
func (d *Disk) serveNotFoundFromConfig(serving *Serving) {
	// Try serving not found page for http://host/ => /group/project/404.html
	if d.tryNotFound(serving.Writer, serving.Request, serving.Project) == nil {
		return
	}

	// Serve generic not found
	httperrors.Serve404(serving.Writer)
}

func (d *Disk) tryFile(w http.ResponseWriter, r *http.Request, projectName string, subPath ...string) error {
	fullPath, err := d.resolvePath(projectName, subPath...)

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(r.URL.Path) {
			fullPath, err = d.resolvePath(projectName, filepath.Join(subPath...), "index.html")
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
		fullPath, err = d.resolvePath(projectName, strings.TrimSuffix(filepath.Join(subPath...), "/")+".html")
	}

	if err != nil {
		return err
	}

	return d.serveFile(w, r, fullPath)
}

func (d *Disk) tryNotFound(w http.ResponseWriter, r *http.Request, projectName string) error {
	page404, err := d.resolvePath(projectName, "404.html")
	if err != nil {
		return err
	}

	err = d.serveCustomFile(w, r, http.StatusNotFound, page404)
	if err != nil {
		return err
	}
	return nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (d *Disk) resolvePath(projectName string, subPath ...string) (string, error) {
	publicPath := filepath.Join(d.Group, projectName, "public")

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

func (d *Disk) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, origPath string) error {
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

	contentType, err := d.detectContentType(origPath)
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

func (d *Disk) serveFile(w http.ResponseWriter, r *http.Request, origPath string) error {
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

	contentType, err := d.detectContentType(origPath)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, origPath, fi.ModTime(), file)

	return nil
}

// Detect file's content-type either by extension or mime-sniffing.
// Implementation is adapted from Golang's `http.serveContent()`
// See https://github.com/golang/go/blob/902fc114272978a40d2e65c2510a18e870077559/src/net/http/fs.go#L194
func (d *Disk) detectContentType(path string) (string, error) {
	contentType := mime.TypeByExtension(filepath.Ext(path))

	if contentType == "" {
		var buf [512]byte

		file, err := os.Open(path)
		if err != nil {
			return "", err
		}

		defer file.Close()

		// Using `io.ReadFull()` because `file.Read()` may be chunked.
		// Ignoring errors because we don't care if the 512 bytes cannot be read.
		n, _ := io.ReadFull(file, buf[:])
		contentType = http.DetectContentType(buf[:n])
	}

	return contentType, nil
}

func acceptsGZip(r *http.Request) bool {
	if r.Header.Get("Range") != "" {
		return false
	}

	offers := []string{"gzip", "identity"}
	acceptedEncoding := httputil.NegotiateContentEncoding(r, offers)
	return acceptedEncoding == "gzip"
}

func handleGZip(w http.ResponseWriter, r *http.Request, fullPath string) string {
	if !acceptsGZip(r) {
		return fullPath
	}

	gzipPath := fullPath + ".gz"

	// Ensure the .gz file is not a symlink
	if fi, err := os.Lstat(gzipPath); err != nil || !fi.Mode().IsRegular() {
		return fullPath
	}

	w.Header().Set("Content-Encoding", "gzip")

	return gzipPath
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

func endsWithoutHTMLExtension(path string) bool {
	return !strings.HasSuffix(path, ".html")
}

func openNoFollow(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW, 0)
}

// HasAcmeChallenge checks domain directory contains particular acme challenge
func (d *Disk) HasAcmeChallenge(token string) bool {
	if d == nil {
		return false
	}

	if !d.CustomDomain {
		return false
	}

	_, err := d.resolvePath(d.Project, ".well-known/acme-challenge", token)

	// there is an acme challenge on disk
	if err == nil {
		return true
	}

	_, err = d.resolvePath(d.Project, ".well-known/acme-challenge", token, "index.html")

	if err == nil {
		return true
	}

	return false
}
