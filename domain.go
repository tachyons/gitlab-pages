package main

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
	"time"

	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httputil"
)

type locationDirectoryError struct {
	FullPath     string
	RelativePath string
}

func (l *locationDirectoryError) Error() string {
	return "location error accessing directory where file expected"
}

type domain struct {
	Group            string
	Project          string
	Config           *domainConfig
	certificate      *tls.Certificate
	certificateError error
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

func (d *domain) serveFile(w http.ResponseWriter, r *http.Request, origPath string) error {
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

	fmt.Println("Serving", fullPath, "for", r.URL.Path)

	// ServeContent sets Content-Type for us
	http.ServeContent(w, r, origPath, fi.ModTime(), file)
	return nil
}

func (d *domain) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, origPath string) error {
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

	// Serve the file
	fmt.Println("Serving", fullPath, "for", r.URL.Path, "with", code)
	w.WriteHeader(code)

	if r.Method != "HEAD" {
		_, err := io.CopyN(w, file, fi.Size())
		return err
	}

	return nil
}

// Resolve the HTTP request to a path on disk, converting requests for
// directories to requests for index.html inside the directory if appropriate.
func (d *domain) resolvePath(projectName string, subPath ...string) (string, error) {
	publicPath := filepath.Join(d.Group, projectName, "public")

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

func (d *domain) tryNotFound(w http.ResponseWriter, r *http.Request, projectName string) error {
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

func (d *domain) tryFile(w http.ResponseWriter, r *http.Request, projectName, pathSuffix string, subPath ...string) error {
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

func (d *domain) serveFromGroup(w http.ResponseWriter, r *http.Request) {
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

func (d *domain) serveFromConfig(w http.ResponseWriter, r *http.Request) {
	// Try to serve file for http://host/... => /group/project/...
	if d.tryFile(w, r, d.Project, "", r.URL.Path) == nil {
		return
	}

	// Try serving not found page for http://host/ => /group/project/404.html
	if d.tryNotFound(w, r, d.Project) == nil {
		return
	}

	// Serve generic not found
	httperrors.Serve404(w)
}

func (d *domain) ensureCertificate() (*tls.Certificate, error) {
	if d.Config == nil {
		return nil, errors.New("tls certificates can be loaded only for pages with configuration")
	}

	if d.certificate != nil || d.certificateError != nil {
		return d.certificate, d.certificateError
	}

	tls, err := tls.X509KeyPair([]byte(d.Config.Certificate), []byte(d.Config.Key))
	if err != nil {
		d.certificateError = err
		return nil, err
	}

	d.certificate = &tls
	return d.certificate, nil
}

func (d *domain) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d.Config != nil {
		d.serveFromConfig(w, r)
	} else {
		d.serveFromGroup(w, r)
	}
}
