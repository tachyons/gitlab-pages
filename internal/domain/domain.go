package domain

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/sys/unix"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httputil"
)

type locationDirectoryError struct {
	FullPath     string
	RelativePath string
}

type locationFileNoExtensionError struct {
	FullPath string
}

// GroupConfig represents a per-request config for a group domain
type GroupConfig interface {
	IsHTTPSOnly(*http.Request) bool
	HasAccessControl(*http.Request) bool
	IsNamespaceProject(*http.Request) bool
	ProjectID(*http.Request) uint64
	ProjectExists(*http.Request) bool
	ProjectWithSubpath(*http.Request) (string, string, error)
}

// Domain is a domain that gitlab-pages can serve.
type Domain struct {
	Group   string
	Project string

	ProjectConfig *ProjectConfig
	GroupConfig   GroupConfig // handles group domain config

	certificate      *tls.Certificate
	certificateError error
	certificateOnce  sync.Once
}

// String implements Stringer.
func (d *Domain) String() string {
	if d.Group != "" && d.Project != "" {
		return d.Group + "/" + d.Project
	}

	if d.Group != "" {
		return d.Group
	}

	return d.Project
}

func (d *Domain) isCustomDomain() bool {
	if d.isUnconfigured() {
		panic("project config and group config should not be nil at the same time")
	}

	return d.ProjectConfig != nil && d.GroupConfig == nil
}

func (d *Domain) isUnconfigured() bool {
	if d == nil {
		return true
	}

	return d.ProjectConfig == nil && d.GroupConfig == nil
}

func (l *locationDirectoryError) Error() string {
	return "location error accessing directory where file expected"
}

func (l *locationFileNoExtensionError) Error() string {
	return "error accessing a path without an extension"
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

// IsHTTPSOnly figures out if the request should be handled with HTTPS
// only by looking at group and project level config.
func (d *Domain) IsHTTPSOnly(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	// Check custom domain config (e.g. http://example.com)
	if d.isCustomDomain() {
		return d.ProjectConfig.HTTPSOnly
	}

	// Check projects served under the group domain, including the default one
	return d.GroupConfig.IsHTTPSOnly(r)
}

// IsAccessControlEnabled figures out if the request is to a project that has access control enabled
func (d *Domain) IsAccessControlEnabled(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	// Check custom domain config (e.g. http://example.com)
	if d.isCustomDomain() {
		return d.ProjectConfig.AccessControl
	}

	// Check projects served under the group domain, including the default one
	return d.GroupConfig.HasAccessControl(r)
}

// HasAcmeChallenge checks domain directory contains particular acme challenge
func (d *Domain) HasAcmeChallenge(token string) bool {
	if d.isUnconfigured() || !d.isCustomDomain() {
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

// IsNamespaceProject figures out if the request is to a namespace project
func (d *Domain) IsNamespaceProject(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	// If request is to a custom domain, we do not handle it as a namespace project
	// as there can't be multiple projects under the same custom domain
	if d.isCustomDomain() {
		return false
	}

	// Check projects served under the group domain, including the default one
	return d.GroupConfig.IsNamespaceProject(r)
}

// GetID figures out what is the ID of the project user tries to access
func (d *Domain) GetID(r *http.Request) uint64 {
	if d.isUnconfigured() {
		return 0
	}

	if d.isCustomDomain() {
		return d.ProjectConfig.ProjectID
	}

	return d.GroupConfig.ProjectID(r)
}

// HasProject figures out if the project exists that the user tries to access
func (d *Domain) HasProject(r *http.Request) bool {
	if d.isUnconfigured() {
		return false
	}

	if d.isCustomDomain() {
		return true
	}

	return d.GroupConfig.ProjectExists(r)
}

// Detect file's content-type either by extension or mime-sniffing.
// Implementation is adapted from Golang's `http.serveContent()`
// See https://github.com/golang/go/blob/902fc114272978a40d2e65c2510a18e870077559/src/net/http/fs.go#L194
func (d *Domain) detectContentType(path string) (string, error) {
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

func (d *Domain) serveFile(w http.ResponseWriter, r *http.Request, origPath string) error {
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

	if !d.IsAccessControlEnabled(r) {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	contentType, err := d.detectContentType(origPath)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, origPath, fi.ModTime(), file)

	return nil
}

func (d *Domain) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, origPath string) error {
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

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (d *Domain) resolvePath(projectName string, subPath ...string) (string, error) {
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

func (d *Domain) tryNotFound(w http.ResponseWriter, r *http.Request, projectName string) error {
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

func (d *Domain) tryFile(w http.ResponseWriter, r *http.Request, projectName string, subPath ...string) error {
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

func (d *Domain) serveFileFromGroup(w http.ResponseWriter, r *http.Request) bool {
	projectName, subPath, err := d.GroupConfig.ProjectWithSubpath(r)
	if err != nil {
		httperrors.Serve404(w)
		return true
	}

	if d.tryFile(w, r, projectName, subPath) == nil {
		return true
	}

	return false
}

func (d *Domain) serveNotFoundFromGroup(w http.ResponseWriter, r *http.Request) {
	projectName, _, err := d.GroupConfig.ProjectWithSubpath(r)

	if err != nil {
		httperrors.Serve404(w)
		return
	}

	// Try serving custom not-found page
	if d.tryNotFound(w, r, projectName) == nil {
		return
	}

	// Generic 404
	httperrors.Serve404(w)
}

func (d *Domain) serveFileFromConfig(w http.ResponseWriter, r *http.Request) bool {
	// Try to serve file for http://host/... => /group/project/...
	if d.tryFile(w, r, d.Project, r.URL.Path) == nil {
		return true
	}

	return false
}

func (d *Domain) serveNotFoundFromConfig(w http.ResponseWriter, r *http.Request) {
	// Try serving not found page for http://host/ => /group/project/404.html
	if d.tryNotFound(w, r, d.Project) == nil {
		return
	}

	// Serve generic not found
	httperrors.Serve404(w)
}

// EnsureCertificate parses the PEM-encoded certificate for the domain
func (d *Domain) EnsureCertificate() (*tls.Certificate, error) {
	if d.isUnconfigured() || !d.isCustomDomain() {
		return nil, errors.New("tls certificates can be loaded only for pages with configuration")
	}

	d.certificateOnce.Do(func() {
		var cert tls.Certificate
		cert, d.certificateError = tls.X509KeyPair(
			[]byte(d.ProjectConfig.Certificate),
			[]byte(d.ProjectConfig.Key),
		)
		if d.certificateError == nil {
			d.certificate = &cert
		}
	})

	return d.certificate, d.certificateError
}

// ServeFileHTTP implements http.Handler. Returns true if something was served, false if not.
func (d *Domain) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	if d.isUnconfigured() {
		httperrors.Serve404(w)
		return true
	}

	if d.isCustomDomain() {
		return d.serveFileFromConfig(w, r)
	}

	return d.serveFileFromGroup(w, r)
}

// ServeNotFoundHTTP implements http.Handler. Serves the not found pages from the projects.
func (d *Domain) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	if d.isUnconfigured() {
		httperrors.Serve404(w)
		return
	}

	if d.isCustomDomain() {
		d.serveNotFoundFromConfig(w, r)
	} else {
		d.serveNotFoundFromGroup(w, r)
	}
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
