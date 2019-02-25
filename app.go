package main

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	ghandlers "github.com/gorilla/handlers"
	"github.com/rs/cors"
	log "github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/errortracking"
	labmetrics "gitlab.com/gitlab-org/labkit/metrics"
	"gitlab.com/gitlab-org/labkit/monitoring"
	"gitlab.com/lupine/go-mimedb"

	"gitlab.com/gitlab-org/gitlab-pages/internal/acme"
	"gitlab.com/gitlab-org/gitlab-pages/internal/artifact"
	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	headerConfig "gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/handlers"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/netutil"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
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

func (a *theApp) checkAuthenticationIfNotExists(domain *domain.Domain, w http.ResponseWriter, r *http.Request) bool {
	if domain == nil || !domain.HasLookupPath(r) {
		// Only if auth is supported
		if a.Auth.IsAuthSupported() {
			// To avoid user knowing if pages exist, we will force user to login and authorize pages
			if a.Auth.CheckAuthenticationWithoutProject(w, r) {
				return true
			}

			// User is authenticated, show the 404
			if domain != nil {
				domain.ServeNotFoundHTTP(w, r)
			} else {
				httperrors.Serve404(w)
			}

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

	if a.checkAuthenticationIfNotExists(domain, w, r) {
		return true
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
		host, domain, err := a.getHostAndDomain(r)
		if err != nil {
			metrics.DomainsSourceFailures.Inc()
			log.WithError(err).Error("could not fetch domain information from a source")

			httperrors.Serve502(w)
			return
		}

		r = request.WithHostAndDomain(r, host, domain)

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
		headerConfig.AddCustomHeaders(w, a.CustomHeaders)

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
			if a.Auth.CheckAuthentication(w, r, domain.GetProjectID(r)) {
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
			// namespace project is public.
			if domain.IsNamespaceProject(r) {
				if a.Auth.CheckAuthenticationWithoutProject(w, r) {
					return
				}

				domain.ServeNotFoundHTTP(w, r)
				return
			}

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
		err := listenAndServe(fd, httpHandler, a.HTTP2, nil, limiter)
		if err != nil {
			capturingFatal(err, errortracking.WithField("listener", request.SchemeHTTP))
		}
	}()
}

func (a *theApp) listenHTTPSFD(wg *sync.WaitGroup, fd uintptr, httpHandler http.Handler, limiter *netutil.Limiter) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := listenAndServeTLS(fd, a.RootCertificate, a.RootKey, httpHandler, a.ServeTLS, a.InsecureCiphers, a.TLSMinVersion, a.TLSMaxVersion, a.HTTP2, limiter)
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
			err := listenAndServe(fd, proxyHandler, a.HTTP2, nil, limiter)
			if err != nil {
				capturingFatal(err, errortracking.WithField("listener", "http proxy"))
			}
		}(fd)
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

	if config.ClientID != "" {
		a.Auth = auth.New(config.Domain, config.StoreSecret, config.ClientID, config.ClientSecret,
			config.RedirectURI, config.GitLabServer)
	}

	a.Handlers = handlers.New(a.Auth, a.Artifact)

	if config.GitLabServer != "" {
		a.AcmeMiddleware = &acme.Middleware{GitlabURL: config.GitLabServer}
	}

	if len(config.CustomHeaders) != 0 {
		customHeaders, err := headerConfig.ParseHeaderString(config.CustomHeaders)
		if err != nil {
			log.WithError(err).Fatal("Unable to parse header string")
		}
		a.CustomHeaders = customHeaders
	}

	if err := mimedb.LoadTypes(); err != nil {
		log.WithError(err).Warn("Loading extended MIME database failed")
	}

	a.Run()
}

// fatal will log a fatal error and exit.
func fatal(err error, message string) {
	log.WithError(err).Fatal(message)
}
