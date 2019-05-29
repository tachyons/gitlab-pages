package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	mimedb "gitlab.com/lupine/go-mimedb"

	"gitlab.com/gitlab-org/gitlab-pages/internal/admin"
	"gitlab.com/gitlab-org/gitlab-pages/internal/artifact"
	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	"gitlab.com/gitlab-org/gitlab-pages/internal/buildservice"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/netutil"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
	xForwardedProto      = "X-Forwarded-Proto"
	xForwardedHost       = "X-Forwarded-Host"
	xForwardedProtoHTTPS = "https"
)

var (
	corsHandler = cors.New(cors.Options{AllowedMethods: []string{"GET"}})
)

type theApp struct {
	appConfig
	dm           domain.Map
	lock         sync.RWMutex
	Artifact     *artifact.Artifact
	BuildService *buildservice.BuildService
	Auth         *auth.Auth
}

func (a *theApp) isReady() bool {
	return a.dm != nil
}

func (a *theApp) domain(host string) *domain.D {
	host = strings.ToLower(host)
	a.lock.RLock()
	defer a.lock.RUnlock()
	domain, _ := a.dm[host]
	return domain
}

func (a *theApp) ServeTLS(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if ch.ServerName == "" {
		return nil, nil
	}

	if domain := a.domain(ch.ServerName); domain != nil {
		tls, _ := domain.EnsureCertificate()
		return tls, nil
	}

	return nil, nil
}

func (a *theApp) healthCheck(w http.ResponseWriter, r *http.Request, https bool) {
	if a.isReady() {
		w.Write([]byte("success"))
	} else {
		http.Error(w, "not yet ready", http.StatusServiceUnavailable)
	}
}

func (a *theApp) redirectToHTTPS(w http.ResponseWriter, r *http.Request, statusCode int) {
	u := *r.URL
	u.Scheme = "https"
	u.Host = r.Host
	u.User = nil

	http.Redirect(w, r, u.String(), statusCode)
}

func (a *theApp) getHostAndDomain(r *http.Request) (host string, domain *domain.D) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	return host, a.domain(host)
}

// IsAuthSupported checks if pages is running with the authentication support
func (a *theApp) IsAuthSupported() bool {
	return a.Auth != nil
}

func (a *theApp) checkAuthenticationIfNotExists(domain *domain.D, w http.ResponseWriter, r *http.Request) bool {
	if domain == nil || !domain.HasProject(r) {
		// Only if auth is supported
		if a.IsAuthSupported() {
			// To avoid user knowing if pages exist, we will force user to login and authorize pages
			if a.Auth.CheckAuthenticationWithoutProject(w, r) {
				return true
			}

			// User is authenticated, show the 404
			httperrors.Serve404(w)
			return true
		}
	}

	// Without auth, fall back to 404
	if domain == nil {
		httperrors.Serve404(w)
		return true
	}

	return false
}

func (a *theApp) tryAuxiliaryHandlers(w http.ResponseWriter, r *http.Request, https bool, host string, domain *domain.D) bool {
	// short circuit content serving to check for a status page
	if r.RequestURI == a.appConfig.StatusPath {
		a.healthCheck(w, r, https)
		return true
	}

	// Add auto redirect
	if !https && a.RedirectHTTP {
		a.redirectToHTTPS(w, r, http.StatusTemporaryRedirect)
		return true
	}

	// In the event a host is prefixed with the artifact prefix an artifact
	// value is created, and an attempt to proxy the request is made
	if a.Artifact.TryMakeRequest(host, w, r) {
		return true
	}

	if a.IsAuthSupported() && a.BuildService != nil {
		if token, err := a.Auth.GetSessionAccessToken(r); err == nil && token != "" {
			if a.BuildService.TryMakeRequest(host, token, w, r) {
				return true
			}
		}
	}

	if !a.isReady() {
		httperrors.Serve503(w)
		return true
	}

	if a.checkAuthenticationIfNotExists(domain, w, r) {
		return true
	}

	if !https && domain.IsHTTPSOnly(r) {
		a.redirectToHTTPS(w, r, http.StatusMovedPermanently)
		return true
	}

	return false
}

func (a *theApp) serveContent(ww http.ResponseWriter, r *http.Request, https bool) {
	w := newLoggingResponseWriter(ww)
	defer w.Log(r)
	metrics.SessionsActive.Inc()
	defer metrics.SessionsActive.Dec()

	host, domain := a.getHostAndDomain(r)
	if a.Auth.TryAuthenticate(&w, r, a.dm, &a.lock) {
		return
	}

	if a.tryAuxiliaryHandlers(&w, r, https, host, domain) {
		return
	}

	// Only for projects that have access control enabled
	if domain.IsAccessControlEnabled(r) {
		if a.Auth.CheckAuthentication(&w, r, domain.GetID(r)) {
			return
		}
	}

	// Serve static file, applying CORS headers if necessary
	if a.DisableCrossOriginRequests {
		a.serveFileOrNotFound(domain)(&w, r)
	} else {
		corsHandler.ServeHTTP(&w, r, a.serveFileOrNotFound(domain))
	}

	metrics.ProcessedRequests.WithLabelValues(strconv.Itoa(w.status), r.Method).Inc()
}

func (a *theApp) serveFileOrNotFound(domain *domain.D) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileServed := domain.ServeFileHTTP(w, r)

		if !fileServed {
			// We need to trigger authentication flow here if file does not exist to prevent exposing possibly private project existence,
			// because the projects override the paths of the namespace project and they might be private even though
			// namespace project is public.
			if domain.IsNamespaceProject(r) {
				if a.Auth.CheckAuthenticationWithoutProject(w, r) {
					return
				}

				httperrors.Serve404(w)
				return
			}

			domain.ServeNotFoundHTTP(w, r)
		}
	}
}

func (a *theApp) ServeHTTP(ww http.ResponseWriter, r *http.Request) {
	https := r.TLS != nil
	a.serveContent(ww, r, https)
}

func (a *theApp) ServeProxy(ww http.ResponseWriter, r *http.Request) {
	forwardedProto := r.Header.Get(xForwardedProto)
	https := forwardedProto == xForwardedProtoHTTPS

	if forwardedHost := r.Header.Get(xForwardedHost); forwardedHost != "" {
		r.Host = forwardedHost
	}

	a.serveContent(ww, r, https)
}

func (a *theApp) UpdateDomains(dm domain.Map) {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.dm = dm
}

func (a *theApp) Run() {
	var wg sync.WaitGroup

	limiter := netutil.NewLimiter(a.MaxConns)

	// Listen for HTTP
	for _, fd := range a.ListenHTTP {
		wg.Add(1)
		go func(fd uintptr) {
			defer wg.Done()
			err := listenAndServe(fd, a.ServeHTTP, a.HTTP2, nil, limiter)
			if err != nil {
				fatal(err)
			}
		}(fd)
	}

	// Listen for HTTPS
	for _, fd := range a.ListenHTTPS {
		wg.Add(1)
		go func(fd uintptr) {
			defer wg.Done()
			err := listenAndServeTLS(fd, a.RootCertificate, a.RootKey, a.ServeHTTP, a.ServeTLS, a.HTTP2, a.InsecureCiphers, limiter)
			if err != nil {
				fatal(err)
			}
		}(fd)
	}

	// Listen for HTTP proxy requests
	for _, fd := range a.ListenProxy {
		wg.Add(1)
		go func(fd uintptr) {
			defer wg.Done()
			err := listenAndServe(fd, a.ServeProxy, a.HTTP2, nil, limiter)
			if err != nil {
				fatal(err)
			}
		}(fd)
	}

	// Serve metrics for Prometheus
	if a.ListenMetrics != 0 {
		wg.Add(1)
		go func(fd uintptr) {
			defer wg.Done()

			handler := promhttp.HandlerFor(prometheus.DefaultGatherer, promhttp.HandlerOpts{}).ServeHTTP
			err := listenAndServe(fd, handler, false, nil, nil)
			if err != nil {
				fatal(err)
			}
		}(a.ListenMetrics)
	}

	a.listenAdminUnix(&wg)
	a.listenAdminHTTPS(&wg)

	go domain.Watch(a.Domain, a.UpdateDomains, time.Second)

	wg.Wait()
}

func (a *theApp) listenAdminUnix(wg *sync.WaitGroup) {
	fd := a.ListenAdminUnix
	if fd == 0 {
		return
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		l, err := net.FileListener(os.NewFile(fd, "[admin-socket-unix]"))
		if err != nil {
			fatal(fmt.Errorf("failed to listen on FD %d: %v", fd, err))
		}
		defer l.Close()

		if err := admin.NewServer(string(a.AdminToken)).Serve(l); err != nil {
			fatal(err)
		}
	}()
}

func (a *theApp) listenAdminHTTPS(wg *sync.WaitGroup) {
	fd := a.ListenAdminHTTPS
	if fd == 0 {
		return
	}

	cert, err := tls.X509KeyPair(a.AdminCertificate, a.AdminKey)
	if err != nil {
		fatal(err)
	}

	wg.Add(1)
	go func() {
		defer wg.Done()

		l, err := net.FileListener(os.NewFile(fd, "[admin-socket-https]"))
		if err != nil {
			fatal(fmt.Errorf("failed to listen on FD %d: %v", fd, err))
		}
		defer l.Close()

		if err := admin.NewTLSServer(string(a.AdminToken), &cert).Serve(l); err != nil {
			fatal(err)
		}
	}()
}

func runApp(config appConfig) {
	a := theApp{appConfig: config}

	if config.ArtifactsServer != "" {
		a.Artifact = artifact.New(config.ArtifactsServer, config.ArtifactsServerTimeout, config.Domain)
	}

	if config.ClientID != "" {
		a.Auth = auth.New(config.Domain, config.StoreSecret, config.ClientID, config.ClientSecret,
			config.RedirectURI, config.GitLabServer)

		a.BuildService = buildservice.New(config.GitLabServer, config.ArtifactsServerTimeout, config.Domain)
	}

	configureLogging(config.LogFormat, config.LogVerbose)

	if err := mimedb.LoadTypes(); err != nil {
		log.WithError(err).Warn("Loading extended MIME database failed")
	}

	a.Run()
}
