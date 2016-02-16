package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

const xForwardedProto = "X-Forwarded-Proto"
const xForwardedProtoHTTPS = "https"

type theApp struct {
	appConfig
	domains domains
	lock    sync.RWMutex
}

func (a *theApp) domain(host string) *domain {
	host = strings.ToLower(host)
	a.lock.RLock()
	defer a.lock.RUnlock()
	domain, _ := a.domains[host]
	return domain
}

func (a *theApp) ServeTLS(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if ch.ServerName == "" {
		return nil, nil
	}

	if domain := a.domain(ch.ServerName); domain != nil {
		tls, _ := domain.ensureCertificate()
		return tls, nil
	}

	return nil, nil
}

func (a *theApp) serveContent(ww http.ResponseWriter, r *http.Request, https bool) {
	w := newLoggingResponseWriter(ww)
	defer w.Log(r)

	// Add auto redirect
	if https && !a.RedirectHTTP {
		u := *r.URL
		u.Scheme = "https"
		u.Host = r.Host
		u.User = nil

		http.Redirect(&w, r, u.String(), 307)
		return
	}

	domain := a.domain(r.Host)
	if domain == nil {
		http.NotFound(&w, r)
		return
	}

	// Serve static file
	domain.ServeHTTP(&w, r)
}

func (a *theApp) ServeHTTP(ww http.ResponseWriter, r *http.Request) {
	https := r.TLS != nil
	a.serveContent(ww, r, https)
}

func (a *theApp) ServeProxy(ww http.ResponseWriter, r *http.Request) {
	forwardedProto := r.Header.Get(xForwardedProto)
	https := forwardedProto == xForwardedProtoHTTPS

	a.serveContent(ww, r, https)
}

func (a *theApp) UpdateDomains(domains domains) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.domains = domains
}

func (a *theApp) Run() {
	var wg sync.WaitGroup

	if a.ListenHTTP != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := listenAndServe(a.ListenHTTP, a.ServeHTTP, a.HTTP2, nil)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	// Listen for HTTPS
	if a.ListenHTTPS != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := listenAndServeTLS(a.ListenHTTPS, a.RootCertificate, a.RootKey, a.ServeHTTP, a.ServeTLS, a.HTTP2)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	// Listen for HTTP proxy requests
	if a.ListenProxy != 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := listenAndServe(a.ListenProxy, a.ServeProxy, a.HTTP2, nil)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	go watchDomains(a.Domain, a.UpdateDomains, time.Second)

	wg.Wait()
}

func runApp(config appConfig) {
	a := theApp{appConfig: config}
	a.Run()
}
