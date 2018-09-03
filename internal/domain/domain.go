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

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httputil"
)

type locationDirectoryError struct {
	FullPath     string
	RelativePath string
}

type project struct {
	HTTPSOnly     bool
	AccessControl bool
	ID            uint64
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

func (d *D) getProject(r *http.Request) *project {
	// Check default domain config (e.g. http://mydomain.gitlab.io)
	if groupProject := d.projects[strings.ToLower(r.Host)]; groupProject != nil {
		return groupProject
	}

	// Check URLs with multiple projects for a group
	// (e.g. http://group.gitlab.io/projectA and http://group.gitlab.io/projectB)
	split := strings.SplitN(r.URL.Path, "/", 3)
	if len(split) < 2 {
		return nil
	}

	return d.projects[split[1]]
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
	if project := d.getProject(r); project != nil {
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
	if project := d.getProject(r); project != nil {
		return project.AccessControl
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

	project := d.getProject(r)

	if project != nil {
		return project.ID
	}

	return 0
}

func (d *D) serveFile(w http.ResponseWriter, r *http.Request, origPath string) error {
	fullPath := handleGZip(w, r, origPath)

	file, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer file.Close()

	fi, err := file.Stat()
	if err != nil {
		return err
	}

	// Set caching headers
	w.Header().Set("Cache-Control", "max-age=600")
	w.Header().Set("Expires", time.Now().Add(10*time.Minute).Format(time.RFC1123))

	// ServeContent sets Content-Type for us
	http.ServeContent(w, r, origPath, fi.ModTime(), file)
	return nil
}

func (d *D) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, origPath string) error {
	fullPath := handleGZip(w, r, origPath)

	// Open and serve content of file
	file, err := os.Open(fullPath)
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

func (d *D) tryFile(w http.ResponseWriter, r *http.Request, projectName, pathSuffix string, subPath ...string) error {
	fullPath, err := d.resolvePath(projectName, subPath...)

	if locationError, _ := err.(*locationDirectoryError); locationError != nil {
		if endsWithSlash(r.URL.Path) {
			fullPath, err = d.resolvePath(projectName, filepath.Join(subPath...), "index.html")
		} else {
			redirectPath := "//" + r.Host + "/"
			if pathSuffix != "" {
				redirectPath += pathSuffix + "/"
			}
			if locationError.RelativePath != "" {
				redirectPath += strings.TrimPrefix(locationError.RelativePath, "/") + "/"
			}
			http.Redirect(w, r, redirectPath, 302)
			return nil
		}
	}

	if err != nil {
		return err
	}

	return d.serveFile(w, r, fullPath)
}

func (d *D) serveFromGroup(w http.ResponseWriter, r *http.Request) {
	// The Path always contains "/" at the beginning
	split := strings.SplitN(r.URL.Path, "/", 3)

	// Try to serve file for http://group.example.com/subpath/... => /group/subpath/...
	if len(split) >= 2 && d.tryFile(w, r, split[1], split[1], split[2:]...) == nil {
		return
	}

	// Try to serve file for http://group.example.com/... => /group/group.example.com/...
	if r.Host != "" && d.tryFile(w, r, strings.ToLower(r.Host), "", r.URL.Path) == nil {
		return
	}

	// Try serving not found page for http://group.example.com/subpath/ => /group/subpath/404.html
	if len(split) >= 2 && d.tryNotFound(w, r, split[1]) == nil {
		return
	}

	// Try serving not found page for http://group.example.com/ => /group/group.example.com/404.html
	if r.Host != "" && d.tryNotFound(w, r, strings.ToLower(r.Host)) == nil {
		return
	}

	// Serve generic not found
	httperrors.Serve404(w)
}

func (d *D) serveFromConfig(w http.ResponseWriter, r *http.Request) {
	// Try to serve file for http://host/... => /group/project/...
	if d.tryFile(w, r, d.projectName, "", r.URL.Path) == nil {
		return
	}

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

// ServeHTTP implements http.Handler.
func (d *D) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d == nil {
		httperrors.Serve404(w)
		return
	}

	if d.config != nil {
		d.serveFromConfig(w, r)
	} else {
		d.serveFromGroup(w, r)
	}
}

func endsWithSlash(path string) bool {
	return strings.HasSuffix(path, "/")
}
