package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"

	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const xForwardedProto = "X-Forwarded-Proto"
const xForwardedProtoHTTPS = "https"

var (
	corsHandler = cors.New(cors.Options{AllowedMethods: []string{"GET"}})
)

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

	metrics.SessionsActive.Inc()
	defer metrics.SessionsActive.Dec()

	// short circuit content serving to check for a status page
	if r.RequestURI == a.appConfig.StatusPath {
		w.Write([]byte("success"))
		return
	}

	// Add auto redirect
	if !https && a.RedirectHTTP {
		u := *r.URL
		u.Scheme = "https"
		u.Host = r.Host
		u.User = nil

		http.Redirect(&w, r, u.String(), 307)
		return
	}

	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}
	domain := a.domain(host)
	if domain == nil {
		serve404(&w)
		return
	}

	// Serve static file, applying CORS headers if necessary
	if a.DisableCrossOriginRequests {
		domain.ServeHTTP(&w, r)
	} else {
		corsHandler.ServeHTTP(&w, r, domain.ServeHTTP)
	}

	metrics.ProcessedRequests.WithLabelValues(strconv.Itoa(w.status), r.Method).Inc()
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

	// Listen for HTTP
	for _, fd := range a.ListenHTTP {
		wg.Add(1)
		go func(fd uintptr) {
			defer wg.Done()
			err := listenAndServe(fd, a.ServeHTTP, a.HTTP2, nil)
			if err != nil {
				log.Fatal(err)
			}
		}(fd)
	}

	// Listen for HTTPS
	for _, fd := range a.ListenHTTPS {
		wg.Add(1)
		go func(fd uintptr) {
			defer wg.Done()
			err := listenAndServeTLS(fd, a.RootCertificate, a.RootKey, a.ServeHTTP, a.ServeTLS, a.HTTP2)
			if err != nil {
				log.Fatal(err)
			}
		}(fd)
	}

	// Listen for HTTP proxy requests
	for _, fd := range a.ListenProxy {
		wg.Add(1)
		go func(fd uintptr) {
			defer wg.Done()
			err := listenAndServe(fd, a.ServeProxy, a.HTTP2, nil)
			if err != nil {
				log.Fatal(err)
			}
		}(fd)
	}

	// Serve metrics for Prometheus
	if a.ListenMetrics != 0 {
		wg.Add(1)
		go func(fd uintptr) {
			defer wg.Done()

			handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}).ServeHTTP
			err := listenAndServe(fd, handler, false, nil)
			if err != nil {
				log.Fatal(err)
			}
		}(a.ListenMetrics)
	}

	go watchDomains(a.Domain, a.UpdateDomains, time.Second)

	wg.Wait()
}

func runApp(config appConfig) {
	a := theApp{appConfig: config}
	a.Run()
}
