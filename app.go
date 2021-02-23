package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	ghandlers "github.com/gorilla/handlers"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/go-mimedb"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/errortracking"
	labmetrics "gitlab.com/gitlab-org/labkit/metrics"
	"gitlab.com/gitlab-org/labkit/monitoring"

	"gitlab.com/gitlab-org/gitlab-pages/internal/acme"
	"gitlab.com/gitlab-org/gitlab-pages/internal/artifact"
	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	cfg "gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/handlers"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/middleware"
	"gitlab.com/gitlab-org/gitlab-pages/internal/netutil"
	"gitlab.com/gitlab-org/gitlab-pages/internal/rejectmethods"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/zip"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
	"gitlab.com/gitlab-org/gitlab-pages/internal/tlsconfig"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
	xForwardedHost = "X-Forwarded-Host"
)

var (
	corsHandler = cors.New(cors.Options{AllowedMethods: []string{"GET"}})
)

type theApp struct {
	appConfig
	domains        *source.Domains
	Artifact       *artifact.Artifact
	Auth           *auth.Auth
	Handlers       *handlers.Handlers
	AcmeMiddleware *acme.Middleware
	CustomHeaders  http.Header
}

func (a *theApp) isReady() bool {
	return a.domains.IsReady()
}

func (a *theApp) ServeTLS(ch *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if ch.ServerName == "" {
		return nil, nil
	}

	if domain, _ := a.domain(ch.ServerName); domain != nil {
		tls, _ := domain.EnsureCertificate()
		return tls, nil
	}

	return nil, nil
}

func (a *theApp) healthCheck(w http.ResponseWriter, r *http.Request, https bool) {
	if a.isReady() {
		w.Write([]byte("success\n"))
	} else {
		http.Error(w, "not yet ready", http.StatusServiceUnavailable)
	}
}

func (a *theApp) redirectToHTTPS(w http.ResponseWriter, r *http.Request, statusCode int) {
	u := *r.URL
	u.Scheme = request.SchemeHTTPS
	u.Host = r.Host
	u.User = nil

	http.Redirect(w, r, u.String(), statusCode)
}

func (a *theApp) getHostAndDomain(r *http.Request) (string, *domain.Domain, error) {
	host := request.GetHostWithoutPort(r)
	domain, err := a.domain(host)

	return host, domain, err
}

func (a *theApp) domain(host string) (*domain.Domain, error) {
	return a.domains.GetDomain(host)
}

// checkAuthAndServeNotFound performs the auth process if domain can't be found
// the main purpose of this process is to avoid leaking the project existence/not-existence
// by behaving the same if user has no access to the project or if project simply does not exists
func (a *theApp) checkAuthAndServeNotFound(domain *domain.Domain, w http.ResponseWriter, r *http.Request) bool {
	// To avoid user knowing if pages exist, we will force user to login and authorize pages
	if a.Auth.CheckAuthenticationWithoutProject(w, r, domain) {
		return true
	}

	// auth succeeded try to serve the correct 404 page
	domain.ServeNotFoundAuthFailed(w, r)
	return true
}

func (a *theApp) tryAuxiliaryHandlers(w http.ResponseWriter, r *http.Request, https bool, host string, domain *domain.Domain) bool {
	// Add auto redirect
	if !https && a.RedirectHTTP {
		a.redirectToHTTPS(w, r, http.StatusTemporaryRedirect)
		return true
	}

	if a.Handlers.HandleArtifactRequest(host, w, r) {
		return true
	}

	if !a.isReady() {
		httperrors.Serve503(w)
		return true
	}

	if !domain.HasLookupPath(r) {
		// redirect to auth and serve not found
		if a.checkAuthAndServeNotFound(domain, w, r) {
			return true
		}
	}

	if !https && domain.IsHTTPSOnly(r) {
		a.redirectToHTTPS(w, r, http.StatusMovedPermanently)
		return true
	}

	return false
}

// routingMiddleware will determine the host and domain for the request, for
// downstream middlewares to use
func (a *theApp) routingMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// if we could not retrieve a domain from domains source we break the
		// middleware chain and simply respond with 502 after logging this
		host, d, err := a.getHostAndDomain(r)
		if err != nil && !errors.Is(err, domain.ErrDomainDoesNotExist) {
			metrics.DomainsSourceFailures.Inc()
			log.WithError(err).Error("could not fetch domain information from a source")

			httperrors.Serve502(w)
			return
		}

		r = request.WithHostAndDomain(r, host, d)

		handler.ServeHTTP(w, r)
	})
}

// healthCheckMiddleware is serving the application status check
func (a *theApp) healthCheckMiddleware(handler http.Handler) (http.Handler, error) {
	healthCheck := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		a.healthCheck(w, r, request.IsHTTPS(r))
	})

	loggedHealthCheck, err := logging.BasicAccessLogger(healthCheck, a.LogFormat, nil)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == a.appConfig.StatusPath {
			loggedHealthCheck.ServeHTTP(w, r)
			return
		}

		handler.ServeHTTP(w, r)
	}), nil
}

// customHeadersMiddleware will inject custom headers into the response
func (a *theApp) customHeadersMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		middleware.AddCustomHeaders(w, a.CustomHeaders)

		handler.ServeHTTP(w, r)
	})
}

// acmeMiddleware will handle ACME challenges
func (a *theApp) acmeMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := request.GetDomain(r)

		if a.AcmeMiddleware.ServeAcmeChallenges(w, r, domain) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// authMiddleware handles authentication requests
func (a *theApp) authMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if a.Auth.TryAuthenticate(w, r, a.domains) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// auxiliaryMiddleware will handle status updates, not-ready requests and other
// not static-content responses
func (a *theApp) auxiliaryMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := request.GetHost(r)
		domain := request.GetDomain(r)
		https := request.IsHTTPS(r)

		if a.tryAuxiliaryHandlers(w, r, https, host, domain) {
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// accessControlMiddleware will handle authorization
func (a *theApp) accessControlMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		domain := request.GetDomain(r)

		// Only for projects that have access control enabled
		if domain.IsAccessControlEnabled(r) {
			// accessControlMiddleware
			if a.Auth.CheckAuthentication(w, r, domain) {
				return
			}
		}

		handler.ServeHTTP(w, r)
	})
}

// serveFileOrNotFoundHandler will serve static content or
// return a 404 Not Found response
func (a *theApp) serveFileOrNotFoundHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		defer metrics.ServingTime.Observe(time.Since(start).Seconds())

		domain := request.GetDomain(r)
		fileServed := domain.ServeFileHTTP(w, r)

		if !fileServed {
			// We need to trigger authentication flow here if file does not exist to prevent exposing possibly private project existence,
			// because the projects override the paths of the namespace project and they might be private even though
			// namespace project is public
			if domain.IsNamespaceProject(r) {
				if a.Auth.CheckAuthenticationWithoutProject(w, r, domain) {
					return
				}
			}

			// domain found and authentication succeeds
			domain.ServeNotFoundHTTP(w, r)
		}
	})
}

// httpInitialMiddleware sets up HTTP requests
func (a *theApp) httpInitialMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handler.ServeHTTP(w, setRequestScheme(r))
	})
}

// proxyInitialMiddleware sets up proxy requests
func (a *theApp) proxyInitialMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if forwardedHost := r.Header.Get(xForwardedHost); forwardedHost != "" {
			r.Host = forwardedHost
		}

		handler.ServeHTTP(w, r)
	})
}

// setRequestScheme will update r.URL.Scheme if empty based on r.TLS
func setRequestScheme(r *http.Request) *http.Request {
	if r.URL.Scheme == request.SchemeHTTPS || r.TLS != nil {
		// make sure is set for non-proxy requests
		r.URL.Scheme = request.SchemeHTTPS
	} else {
		r.URL.Scheme = request.SchemeHTTP
	}

	return r
}

func (a *theApp) buildHandlerPipeline() (http.Handler, error) {
	// Handlers should be applied in a reverse order
	handler := a.serveFileOrNotFoundHandler()
	if !a.DisableCrossOriginRequests {
		handler = corsHandler.Handler(handler)
	}
	handler = a.accessControlMiddleware(handler)
	handler = a.auxiliaryMiddleware(handler)
	handler = a.authMiddleware(handler)
	handler = a.acmeMiddleware(handler)
	handler, err := logging.AccessLogger(handler, a.LogFormat)
	if err != nil {
		return nil, err
	}

	// Metrics
	metricsMiddleware := labmetrics.NewHandlerFactory(labmetrics.WithNamespace("gitlab_pages"))
	handler = metricsMiddleware(handler)

	handler = a.routingMiddleware(handler)

	// Health Check
	handler, err = a.healthCheckMiddleware(handler)
	if err != nil {
		return nil, err
	}

	// Custom response headers
	handler = a.customHeadersMiddleware(handler)

	// Correlation ID injection middleware
	var correlationOpts []correlation.InboundHandlerOption
	if a.appConfig.PropagateCorrelationID {
		correlationOpts = append(correlationOpts, correlation.WithPropagation())
	}
	handler = correlation.InjectCorrelationID(handler, correlationOpts...)

	// This MUST be the last handler!
	// This handler blocks unknown HTTP methods,
	// being the last means it will be evaluated first
	// preventing any operation on bogus requests.
	handler = rejectmethods.NewMiddleware(handler)

	return handler, nil
}

func (a *theApp) Run() {
	var wg sync.WaitGroup

	limiter := netutil.NewLimiter(a.MaxConns)

	// Use a common pipeline to use a single instance of each handler,
	// instead of making two nearly identical pipelines
	commonHandlerPipeline, err := a.buildHandlerPipeline()
	if err != nil {
		log.WithError(err).Fatal("Unable to configure pipeline")
	}

	proxyHandler := a.proxyInitialMiddleware(ghandlers.ProxyHeaders(commonHandlerPipeline))

	httpHandler := a.httpInitialMiddleware(commonHandlerPipeline)

	// Listen for HTTP
	for _, fd := range a.ListenHTTP {
		a.listenHTTPFD(&wg, fd, httpHandler, limiter)
	}

	// Listen for HTTPS
	for _, fd := range a.ListenHTTPS {
		a.listenHTTPSFD(&wg, fd, httpHandler, limiter)
	}

	// Listen for HTTP proxy requests
	for _, fd := range a.ListenProxy {
		a.listenProxyFD(&wg, fd, proxyHandler, limiter)
	}

	// Listen for HTTPS PROXYv2 requests
	for _, fd := range a.ListenHTTPSProxyv2 {
		a.ListenHTTPSProxyv2FD(&wg, fd, httpHandler, limiter)
	}

	// Serve metrics for Prometheus
	if a.ListenMetrics != 0 {
		a.listenMetricsFD(&wg, a.ListenMetrics)
	}

	a.domains.Read(a.Domain)

	wg.Wait()
}

func (a *theApp) listenHTTPFD(wg *sync.WaitGroup, fd uintptr, httpHandler http.Handler, limiter *netutil.Limiter) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := listenAndServe(fd, httpHandler, a.HTTP2, nil, limiter, false)
		if err != nil {
			capturingFatal(err, errortracking.WithField("listener", request.SchemeHTTP))
		}
	}()
}

func (a *theApp) listenHTTPSFD(wg *sync.WaitGroup, fd uintptr, httpHandler http.Handler, limiter *netutil.Limiter) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		tlsConfig, err := a.TLSConfig()
		if err != nil {
			capturingFatal(err, errortracking.WithField("listener", request.SchemeHTTPS))
		}

		err = listenAndServe(fd, httpHandler, a.HTTP2, tlsConfig, limiter, false)
		if err != nil {
			capturingFatal(err, errortracking.WithField("listener", request.SchemeHTTPS))
		}
	}()
}

func (a *theApp) listenProxyFD(wg *sync.WaitGroup, fd uintptr, proxyHandler http.Handler, limiter *netutil.Limiter) {
	wg.Add(1)
	go func() {
		wg.Add(1)
		go func(fd uintptr) {
			defer wg.Done()
			err := listenAndServe(fd, proxyHandler, a.HTTP2, nil, limiter, false)
			if err != nil {
				capturingFatal(err, errortracking.WithField("listener", "http proxy"))
			}
		}(fd)
	}()
}

// https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt
func (a *theApp) ListenHTTPSProxyv2FD(wg *sync.WaitGroup, fd uintptr, httpHandler http.Handler, limiter *netutil.Limiter) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		tlsConfig, err := a.TLSConfig()
		if err != nil {
			capturingFatal(err, errortracking.WithField("listener", request.SchemeHTTPS))
		}

		err = listenAndServe(fd, httpHandler, a.HTTP2, tlsConfig, limiter, true)
		if err != nil {
			capturingFatal(err, errortracking.WithField("listener", request.SchemeHTTPS))
		}
	}()
}

func (a *theApp) listenMetricsFD(wg *sync.WaitGroup, fd uintptr) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		l, err := net.FileListener(os.NewFile(fd, "[socket]"))
		if err != nil {
			capturingFatal(fmt.Errorf("failed to listen on FD %d: %v", fd, err), errortracking.WithField("listener", "metrics"))
		}

		monitoringOpts := []monitoring.Option{
			monitoring.WithBuildInformation(VERSION, ""),
			monitoring.WithListener(l),
		}

		err = monitoring.Start(monitoringOpts...)
		if err != nil {
			capturingFatal(err, errortracking.WithField("listener", "metrics"))
		}
	}()
}

func runApp(config appConfig) {
	domains, err := source.NewDomains(config)
	if err != nil {
		log.WithError(err).Fatal("could not create domains config source")
	}

	a := theApp{appConfig: config, domains: domains}

	err = logging.ConfigureLogging(a.LogFormat, a.LogVerbose)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize logging")
	}

	if config.ArtifactsServer != "" {
		a.Artifact = artifact.New(config.ArtifactsServer, config.ArtifactsServerTimeout, config.Domain)
	}

	a.setAuth(config)

	a.Handlers = handlers.New(a.Auth, a.Artifact)

	if config.GitLabServer != "" {
		a.AcmeMiddleware = &acme.Middleware{GitlabURL: config.GitLabServer}
	}

	if len(config.CustomHeaders) != 0 {
		customHeaders, err := middleware.ParseHeaderString(config.CustomHeaders)
		if err != nil {
			log.WithError(err).Fatal("Unable to parse header string")
		}
		a.CustomHeaders = customHeaders
	}

	if err := mimedb.LoadTypes(); err != nil {
		log.WithError(err).Warn("Loading extended MIME database failed")
	}

	c := &cfg.Config{
		Zip: &cfg.ZipServing{
			ExpirationInterval: config.ZipCacheExpiry,
			CleanupInterval:    config.ZipCacheCleanup,
			RefreshInterval:    config.ZipCacheRefresh,
			OpenTimeout:        config.ZipeOpenTimeout,
			AllowedPaths:       []string{config.PagesRoot},
		},
	}

	// TODO: reconfigure all VFS'
	//  https://gitlab.com/gitlab-org/gitlab-pages/-/issues/512
	if err := zip.Instance().Reconfigure(c); err != nil {
		fatal(err, "failed to reconfigure zip VFS")
	}

	a.Run()
}

func (a *theApp) setAuth(config appConfig) {
	if config.ClientID == "" {
		return
	}

	var err error
	a.Auth, err = auth.New(config.Domain, config.StoreSecret, config.ClientID, config.ClientSecret,
		config.RedirectURI, config.GitLabServer, config.AuthScope)
	if err != nil {
		log.WithError(err).Fatal("could not initialize auth package")
	}
}

// fatal will log a fatal error and exit.
func fatal(err error, message string) {
	log.WithError(err).Fatal(message)
}

func (a *theApp) TLSConfig() (*tls.Config, error) {
	return tlsconfig.Create(a.RootCertificate, a.RootKey, a.ServeTLS,
		a.InsecureCiphers, a.TLSMinVersion, a.TLSMaxVersion)
}
