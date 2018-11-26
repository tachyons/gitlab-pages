package domain

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"mime"
	"net"
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

type project struct {
	NamespaceProject bool
	HTTPSOnly        bool
	AccessControl    bool
	ID               uint64
}

type projects map[string]*project

// D is a domain that gitlab-pages can serve.
type D struct {
	group string

	// custom domains:
	projectName string
	config      *domainConfig

	certificate      *tls.Certificate
	certificateError error
	certificateOnce  sync.Once

	// group domains:
	projects projects
}

// String implements Stringer.
func (d *D) String() string {
	if d.group != "" && d.projectName != "" {
		return d.group + "/" + d.projectName
	}

	if d.group != "" {
		return d.group
	}

	return d.projectName
}

func (l *locationDirectoryError) Error() string {
	return "location error accessing directory where file expected"
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

func setContentType(w http.ResponseWriter, fullPath string) {
	ext := filepath.Ext(fullPath)
	ctype := mime.TypeByExtension(ext)
	if ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
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
func (d *D) getProjectWithSubpath(r *http.Request) (*project, string, string) {
	// Check for a project specified in the URL: http://group.gitlab.io/projectA
	// If present, these projects shadow the group domain.
	split := strings.SplitN(r.URL.Path, "/", 3)
	if len(split) >= 2 {
		projectName := strings.ToLower(split[1])
		if project := d.projects[projectName]; project != nil {
			return project, split[1], strings.Join(split[2:], "/")
		}
	}

	// Since the URL doesn't specify a project (e.g. http://mydomain.gitlab.io),
	// return the group project if it exists.
	if host := getHost(r); host != "" {
		if groupProject := d.projects[host]; groupProject != nil {
			return groupProject, host, strings.Join(split[1:], "/")
		}
	}

	return nil, "", ""
}

// IsHTTPSOnly figures out if the request should be handled with HTTPS
// only by looking at group and project level config.
func (d *D) IsHTTPSOnly(r *http.Request) bool {
	if d == nil {
		return false
	}

	// Check custom domain config (e.g. http://example.com)
	if d.config != nil {
		return d.config.HTTPSOnly
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

	// Check custom domain config (e.g. http://example.com)
	if d.config != nil {
		return d.config.AccessControl
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

	// If request is to a custom domain, we do not handle it as a namespace project
	// as there can't be multiple projects under the same custom domain
	if d.config != nil {
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

	if d.config != nil {
		return d.config.ID
	}

	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return project.ID
	}

	return 0
}

// HasProject figures out if the project exists that the user tries to access
func (d *D) HasProject(r *http.Request) bool {
	if d == nil {
		return false
	}

	if d.config != nil {
		return true
	}

	if project, _, _ := d.getProjectWithSubpath(r); project != nil {
		return true
	}

	return false
}

func (d *D) serveFile(w http.ResponseWriter, r *http.Request, origPath string) error {
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

	// ServeContent sets Content-Type for us
	http.ServeContent(w, r, origPath, fi.ModTime(), file)
	return nil
}

func (d *D) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, origPath string) error {
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

	setContentType(w, origPath)
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
func (d *D) resolvePath(projectName string, subPath ...string) (string, error) {
	publicPath := filepath.Join(d.group, projectName, "public")

	// Don't use filepath.Join as cleans the path,
	// where we want to traverse full path as supplied by user
	// (including ..)
	testPath := publicPath + "/" + strings.Join(subPath, "/")
	fullPath, err := filepath.EvalSymlinks(testPath)
	if err != nil {
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

func (d *D) tryNotFound(w http.ResponseWriter, r *http.Request, projectName string) error {
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

func (d *D) tryFile(w http.ResponseWriter, r *http.Request, projectName string, subPath ...string) error {
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

	if err != nil {
		return err
	}

	return d.serveFile(w, r, fullPath)
}

func (d *D) serveFileFromGroup(w http.ResponseWriter, r *http.Request) bool {
	project, projectName, subPath := d.getProjectWithSubpath(r)
	if project == nil {
		httperrors.Serve404(w)
		return true
	}

	if d.tryFile(w, r, projectName, subPath) == nil {
		return true
	}

	return false
}

func (d *D) serveNotFoundFromGroup(w http.ResponseWriter, r *http.Request) {
	project, projectName, _ := d.getProjectWithSubpath(r)
	if project == nil {
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

func (d *D) serveFileFromConfig(w http.ResponseWriter, r *http.Request) bool {
	// Try to serve file for http://host/... => /group/project/...
	if d.tryFile(w, r, d.projectName, r.URL.Path) == nil {
		return true
	}

	return false
}

func (d *D) serveNotFoundFromConfig(w http.ResponseWriter, r *http.Request) {
	// Try serving not found page for http://host/ => /group/project/404.html
	if d.tryNotFound(w, r, d.projectName) == nil {
		return
	}

	// Serve generic not found
	httperrors.Serve404(w)
}

// EnsureCertificate parses the PEM-encoded certificate for the domain
func (d *D) EnsureCertificate() (*tls.Certificate, error) {
	if d.config == nil {
		return nil, errors.New("tls certificates can be loaded only for pages with configuration")
	}

	d.certificateOnce.Do(func() {
		var cert tls.Certificate
		cert, d.certificateError = tls.X509KeyPair([]byte(d.config.Certificate), []byte(d.config.Key))
		if d.certificateError == nil {
			d.certificate = &cert
		}
	})

	return d.certificate, d.certificateError
}

// ServeFileHTTP implements http.Handler. Returns true if something was served, false if not.
func (d *D) ServeFileHTTP(w http.ResponseWriter, r *http.Request) bool {
	if d == nil {
		httperrors.Serve404(w)
		return true
	}

	if d.config != nil {
		return d.serveFileFromConfig(w, r)
	}

	return d.serveFileFromGroup(w, r)
}

// ServeNotFoundHTTP implements http.Handler. Serves the not found pages from the projects.
func (d *D) ServeNotFoundHTTP(w http.ResponseWriter, r *http.Request) {
	if d == nil {
		httperrors.Serve404(w)
		return
	}

	if d.config != nil {
		d.serveNotFoundFromConfig(w, r)
	} else {
		d.serveNotFoundFromGroup(w, r)
	}
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}

func openNoFollow(path string) (*os.File, error) {
	return os.OpenFile(path, os.O_RDONLY|unix.O_NOFOLLOW, 0)
}
