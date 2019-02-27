package domain

import (
	"crypto/tls"
	"fmt"
	"io"
	"mime"
	"net"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/client"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httputil"
	"gitlab.com/gitlab-org/gitlab-pages/internal/storage"
)

const (
	subgroupScanLimit int = 21
	// maxProjectDepth is set to the maximum nested project depth in gitlab (21) plus 3.
	// One for the project, one for the first empty element of the split (URL.Path starts with /),
	// and one for the real file path
	maxProjectDepth int = subgroupScanLimit + 3
)

type locationDirectoryError struct {
	FullPath     string
	RelativePath string
}

type locationFileNoExtensionError struct {
	FullPath string
}

type project struct {
	NamespaceProject bool
	HTTPSOnly        bool
	AccessControl    bool
	ID               uint64
}

// D is a domain that gitlab-pages can serve.
type D struct {
	*client.DomainResponse
}

// Finder provides a mapping between host and domain configuration
type Finder func(host string) *D

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

func (d *D) handleGZip(w http.ResponseWriter, r *http.Request, storage storage.S, fullPath string) string {
	if !acceptsGZip(r) {
		return fullPath
	}

	gzipPath := fullPath + ".gz"

	// Ensure the .gz file is not a symlink
	if fi, err := storage.Stat(gzipPath); err != nil || !fi.Mode().IsRegular() {
		return fullPath
	}

	w.Header().Set("Content-Encoding", "gzip")

	return gzipPath
}

func getHost(r *http.Request) string {
	host := strings.ToLower(r.Host)

	if splitHost, _, err := net.SplitHostPort(host); err == nil {
		host = splitHost
	}

	return host
}

// Look up a project inside the domain based on the host and path. Returns the
// project and its name (if applicable)
func (d *D) getProjectWithSubpath(r *http.Request) (*client.LookupPath, string, string) {
	lp, err := d.DomainResponse.GetPath(r.URL.Path)
	if err != nil {
		return nil, "", ""
	}

	return lp, "", lp.Tail(r.URL.Path)
}

// IsHTTPSOnly figures out if the request should be handled with HTTPS
// only by looking at group and project level config.
func (d *D) IsHTTPSOnly(r *http.Request) bool {
	if d == nil {
		return false
	}

	// Check projects served under the group domain, including the default one
	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return project.HTTPSOnly
	}

	return false
}

// IsAccessControlEnabled figures out if the request is to a project that has access control enabled
func (d *D) IsAccessControlEnabled(r *http.Request) bool {
	if d == nil {
		return false
	}

	// Check projects served under the group domain, including the default one
	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return project.AccessControl
	}

	return false
}

// IsNamespaceProject figures out if the request is to a namespace project
func (d *D) IsNamespaceProject(r *http.Request) bool {
	if d == nil {
		return false
	}

	// Check projects served under the group domain, including the default one
	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return project.NamespaceProject
	}

	return false
}

// GetID figures out what is the ID of the project user tries to access
func (d *D) GetID(r *http.Request) uint64 {
	if d == nil {
		return 0
	}

	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return project.ProjectID
	}

	return 0
}

// HasProject figures out if the project exists that the user tries to access
func (d *D) HasProject(r *http.Request) bool {
	if d == nil {
		return false
	}

	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return true
	}

	return false
}

// Detect file's content-type either by extension or mime-sniffing.
// Implementation is adapted from Golang's `http.serveContent()`
// See https://github.com/golang/go/blob/902fc114272978a40d2e65c2510a18e870077559/src/net/http/fs.go#L194
func (d *D) detectContentType(storage storage.S, path string) (string, error) {
	contentType := mime.TypeByExtension(filepath.Ext(path))

	if contentType == "" {
		var buf [512]byte

		file, _, err := storage.Open(path)
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

func (d *D) serveFile(w http.ResponseWriter, r *http.Request, storage storage.S, origPath string) error {
	fullPath := d.handleGZip(w, r, storage, origPath)

	file, fi, err := storage.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if !d.IsAccessControlEnabled(r) {
		// Set caching headers
		w.Header().Set("Cache-Control", "max-age=600")
		w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))
	}

	contentType, err := d.detectContentType(storage, origPath)
	if err != nil {
		return err
	}

	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, origPath, fi.ModTime(), file)

	return nil
}

func (d *D) serveCustomFile(w http.ResponseWriter, r *http.Request, storage storage.S, code int, origPath string) error {
	fullPath := d.handleGZip(w, r, storage, origPath)

	// Open and serve content of file
	file, fi, err := storage.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	contentType, err := d.detectContentType(storage, origPath)
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
func (d *D) resolvePath(storage storage.S, subPath ...string) (string, error) {
	fullPath, err := storage.Resolve(strings.Join(subPath, "/"))
	if err != nil {
		if endsWithoutHTMLExtension(fullPath) {
			return "", &locationFileNoExtensionError{
				FullPath: fullPath,
			}
		}

		return "", err
	}

	fi, err := storage.Stat(fullPath)
	if err != nil {
		return "", err
	}

	// The requested path is a directory, so try index.html via recursion
	if fi.IsDir() {
		return "", &locationDirectoryError{
			FullPath: fullPath,
		}
	}

	// The file exists, but is not a supported type to serve. Perhaps a block
	// special device or something else that may be a security risk.
	if !fi.Mode().IsRegular() {
		return "", fmt.Errorf("%s: is not a regular file", fullPath)
	}

	return fullPath, nil
}

func (d *D) tryNotFound(w http.ResponseWriter, r *http.Request, storage storage.S) error {
	page404, err := d.resolvePath(storage, "404.html")
	if err != nil {
		return err
	}

	err = d.serveCustomFile(w, r, storage, http.StatusNotFound, page404)
	if err != nil {
		return err
	}
	return nil
}

func (d *D) tryFile(w http.ResponseWriter, r *http.Request, storage storage.S, subPath ...string) error {
	fullPath, err := d.resolvePath(storage, subPath...)

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(r.URL.Path) {
			fullPath, err = d.resolvePath(storage, filepath.Join(subPath...), "index.html")
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
		fullPath, err = d.resolvePath(storage, strings.TrimSuffix(filepath.Join(subPath...), "/")+".html")
	}

	if err != nil {
		return err
	}

	return d.serveFile(w, r, storage, fullPath)
}

// Certificate parses the PEM-encoded certificate for the domain
func (d *D) Certificate() (tls.Certificate, error) {
	return tls.X509KeyPair([]byte(d.DomainResponse.Certificate), []byte(d.DomainResponse.Key))
}

// ServeFileHTTP implements http.Handler. Returns true if something was served, false if not.
func (d *D) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	if d == nil {
		httperrors.Serve404(w)
		return true
	}

	project, _, subPath := d.getProjectWithSubpath(r)
	if project == nil {
		httperrors.Serve404(w)
		return true
	}

	if d.tryFile(w, r, storage.New(project), subPath) == nil {
		return true
	}

	return false
}

// ServeNotFoundHTTP implements http.Handler. Serves the not found pages from the projects.
func (d *D) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	if d == nil {
		httperrors.Serve404(w)
		return
	}

	project, _, _ := d.getProjectWithSubpath(r)
	if project == nil {
		httperrors.Serve404(w)
		return
	}

	// Try serving custom not-found page
	if d.tryNotFound(w, r, storage.New(project)) == nil {
		return
	}

	// Generic 404
	httperrors.Serve404(w)
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

func endsWithoutHTMLExtension(path string) bool {
	return !strings.HasSuffix(path, ".html")
}
