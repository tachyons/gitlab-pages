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
)

type domain struct {
	Group            string
	Project          string
	Config           *domainConfig
	certificate      *tls.Certificate
	certificateError error
}

func (d *domain) serveFile(w http.ResponseWriter, r *http.Request, fullPath string) error {
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

	println("Serving", fullPath, "for", r.URL.Path)
	http.ServeContent(w, r, filepath.Base(file.Name()), fi.ModTime(), file)
	return nil
}

func (d *domain) serveCustomFile(w http.ResponseWriter, r *http.Request, code int, fullPath string) error {
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

	println("Serving", fullPath, "for", r.URL.Path, "with", code)

	// Serve the file
	_, haveType := w.Header()["Content-Type"]
	if !haveType {
		ctype := mime.TypeByExtension(filepath.Ext(fullPath))
		w.Header().Set("Content-Type", ctype)
	}
	w.Header().Set("Content-Length", strconv.FormatInt(fi.Size(), 10))
	w.WriteHeader(code)
	if r.Method != "HEAD" {
		io.CopyN(w, file, fi.Size())
	}
	return nil
}

func (d *domain) resolvePath(projectName, subPath string) (fullPath string, err error) {
	publicPath := filepath.Join(d.Group, projectName, "public")

	fullPath = filepath.Join(publicPath, subPath)
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
	case fi.IsDir() && !strings.HasSuffix(r.URL.Path, "/"):
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

func (d *domain) tryFile(w http.ResponseWriter, r *http.Request, projectName, subPath string) error {
	path, err := d.resolvePath(projectName, subPath)
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
	if len(split) >= 2 {
		subPath := ""
		if len(split) >= 3 {
			subPath = split[2]
		}
		if d.tryFile(w, r, split[1], subPath) == nil {
			return
		}
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
	if r.Host != "" && d.tryFile(w, r, strings.ToLower(r.Host), r.URL.Path) == nil {
		return
	}

	// Serve generic not found
	http.NotFound(w, r)
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

	http.NotFound(w, r)
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
