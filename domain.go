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

func (d *domain) resolvePath(projectName string, subPath ...string) (fullPath string, err error) {
	publicPath := filepath.Join(d.Group, projectName, "public")

	fullPath = filepath.Join(publicPath, filepath.Join(subPath...))
	fullPath, err = filepath.EvalSymlinks(fullPath)
	if err != nil {
		return
	}

	if !strings.HasPrefix(fullPath, publicPath+"/") && fullPath != publicPath {
		err = fmt.Errorf("%q should be in %q", fullPath, publicPath)
		return
	}
	return
}

func (d *domain) tryNotFound(w http.ResponseWriter, r *http.Request, projectName string) error {
	page404, err := d.resolvePath(projectName, "404.html")
	if err != nil {
		return err
	}

	// Make sure that file is not symlink
	fi, err := os.Lstat(page404)
	if err != nil && !fi.Mode().IsRegular() {
		return err
	}

	err = d.serveCustomFile(w, r, http.StatusNotFound, page404)
	if err != nil {
		return err
	}
	return nil
}

func (d *domain) checkPath(w http.ResponseWriter, r *http.Request, path string) (fullPath string, err error) {
	fullPath = path
	fi, err := os.Lstat(fullPath)
	if err != nil {
		return
	}

	switch {
	// If the URL doesn't end with /, send location to client
	case fi.IsDir() && !endsWithSlash(r.URL.Path):
		newURL := *r.URL
		newURL.Path += "/"
		http.Redirect(w, r, newURL.String(), 302)

		// If this is directory, we try the index.html
	case fi.IsDir():
		fullPath = filepath.Join(fullPath, "index.html")
		fi, err = os.Lstat(fullPath)
		if err != nil {
			return
		}

		// We don't allow to open the regular file
	case !fi.Mode().IsRegular():
		err = fmt.Errorf("%s: is not a regular file", fullPath)
	}
	return
}

func (d *domain) tryFile(w http.ResponseWriter, r *http.Request, projectName string, subPath ...string) error {
	path, err := d.resolvePath(projectName, subPath...)
	if err != nil {
		return err
	}
	path, err = d.checkPath(w, r, path)
	if err != nil {
		return err
	}
	return d.serveFile(w, r, path)
}

func (d *domain) serveFromGroup(w http.ResponseWriter, r *http.Request) {
	// The Path always contains "/" at the beginning
	split := strings.SplitN(r.URL.Path, "/", 3)

	// Try to serve file for http://group.example.com/subpath/... => /group/subpath/...
	if len(split) >= 2 && d.tryFile(w, r, split[1], split[2:]...) == nil {
		return
	}

	// Try to serve file for http://group.example.com/... => /group/group.example.com/...
	if r.Host != "" && d.tryFile(w, r, strings.ToLower(r.Host), r.URL.Path) == nil {
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
	if d.tryFile(w, r, d.Project, r.URL.Path) == nil {
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
