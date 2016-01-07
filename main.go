package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
)

var listenHTTP = flag.String("listen-http", ":80", "The address to listen for HTTP requests")
var listenHTTPS = flag.String("listen-https", "", "The address to listen for HTTPS requests")
var pagesDomain = flag.String("pages-domain", "gitlab-example.com", "The domain to serve static pages")
var pagesRootCert = flag.String("root-cert", "", "The default certificate to serve static pages")
var pagesRootKey = flag.String("root-key", "", "The default certificate to serve static pages")
var serverHTTP = flag.Bool("serve-http", true, "Serve the pages under HTTP")
var http2proto = flag.Bool("http2", true, "Enable HTTP2 support")
var pagesRoot = flag.String("pages-root", "shared/pages", "The directory where pages are stored")

type theApp struct {
	domains domains
}

func (a *theApp) ServeTLS(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if ch.ServerName == "" {
		return nil, nil
	}

	host := strings.ToLower(ch.ServerName)
	if domain, ok := a.domains[host]; ok {
		tls, _ := domain.ensureCertificate()
		return tls, nil
	}

	return nil, nil
}

func (a *theApp) ServeHTTP(ww http.ResponseWriter, r *http.Request) {
	w := newLoggingResponseWriter(ww)
	defer w.Log(r)

	// Add auto redirect
	if r.TLS == nil && !*serverHTTP {
		u := *r.URL
		u.Scheme = "https"
		u.Host = r.Host
		u.User = nil

		http.Redirect(&w, r, u.String(), 307)
		return
	}

	host := strings.ToLower(r.Host)
	domain, ok := a.domains[host]

	if !ok {
		http.NotFound(&w, r)
		return
	}

	// Serve static file
	domain.ServeHTTP(&w, r)
}

func (a *theApp) UpdateDomains(domains domains) {
	fmt.Printf("Domains: %v", domains)
	a.domains = domains
}

func main() {
	var wg sync.WaitGroup
	var app theApp

	flag.Parse()

	// Listen for HTTP
	if *listenHTTP != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := ListenAndServe(*listenHTTP, &app)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	// Listen for HTTPS
	if *listenHTTPS != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := ListenAndServeTLS(*listenHTTPS, *pagesRootCert, *pagesRootKey, &app)
			if err != nil {
				log.Fatal(err)
			}
		}()
	}

	go watchDomains(app.UpdateDomains)

	wg.Wait()
}
