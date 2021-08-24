package main

import (
	"context"
	cryptotls "crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	ghandlers "github.com/gorilla/handlers"
	"github.com/rs/cors"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/go-mimedb"
	"gitlab.com/gitlab-org/labkit/errortracking"
	labmetrics "gitlab.com/gitlab-org/labkit/metrics"
	"gitlab.com/gitlab-org/labkit/monitoring"

	"gitlab.com/gitlab-org/gitlab-pages/internal/acme"
	"gitlab.com/gitlab-org/gitlab-pages/internal/artifact"
	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	cfg "gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
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
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
	xForwardedHost = "X-Forwarded-Host"

	pathPrefixArtifacts = "/-/"
	pathPrefixAuth      = "/auth"
)

var (
	corsHandler = cors.New(cors.Options{AllowedMethods: []string{http.MethodGet, http.MethodHead}})
)

type theApp struct {
	config         *cfg.Config
	domains        *source.Domains
	Artifact       *artifact.Artifact
	Auth           *auth.Auth
	Handlers       *handlers.Handlers
	AcmeMiddleware *acme.Middleware
	CustomHeaders  http.Header
	knownPaths     []string
}

func (a *theApp) isReady() bool {
	return a.domains.IsReady()
}

func (a *theApp) ServeTLS(ch *cryptotls.ClientHelloInfo) (*cryptotls.Certificate, error) {
	if ch.ServerName == "" {
		return nil, nil
	}

	if domain, _ := a.domain(context.Background(), ch.ServerName); domain != nil {
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
	// do not fetch domain's config if it's a known path || host
	for _, path := range a.knownPaths {
		if strings.HasPrefix(r.URL.Path, path) {
			return host, nil, nil
		}
	}

	if host == a.config.General.Domain {
		return host, nil, nil
	}

	domain, err := a.domain(r.Context(), host)

	return host, domain, err
}

func (a *theApp) domain(ctx context.Context, host string) (*domain.Domain, error) {
	return a.domains.GetDomain(ctx, host)
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
	if !https && a.config.General.RedirectHTTP {
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

	if _, err := domain.GetLookupPath(r); err != nil {
		if errors.Is(err, gitlab.ErrDiskDisabled) {
			errortracking.Capture(err)
			httperrors.Serve500(w)
			return true
		}

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
			logging.LogRequest(r).WithError(err).Error("could not fetch domain information from a source")

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

	loggedHealthCheck, err := logging.BasicAccessLogger(healthCheck, a.config.Log.Format, nil)
	if err != nil {
		return nil, err
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.RequestURI == a.config.General.StatusPath {
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
		fmt.Printf("authMiddleware do we have a correlation ID yet? %q\n", correlation.ExtractFromContext(r.Context()))
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
	if !a.config.General.DisableCrossOriginRequests {
		handler = corsHandler.Handler(handler)
	}
	handler = a.accessControlMiddleware(handler)
	handler = a.auxiliaryMiddleware(handler)
	handler = a.authMiddleware(handler)
	handler = a.acmeMiddleware(handler)
	handler, err := logging.AccessLogger(handler, a.config.Log.Format)
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
	if a.config.General.PropagateCorrelationID {
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

	var limiter *netutil.Limiter
	if a.config.General.MaxConns > 0 {
		limiter = netutil.NewLimiterWithMetrics(
			a.config.General.MaxConns,
			metrics.LimitListenerMaxConns,
			metrics.LimitListenerConcurrentConns,
			metrics.LimitListenerWaitingConns,
		)
	}

	// Use a common pipeline to use a single instance of each handler,
	// instead of making two nearly identical pipelines
	commonHandlerPipeline, err := a.buildHandlerPipeline()
	if err != nil {
		log.WithError(err).Fatal("Unable to configure pipeline")
	}

	proxyHandler := a.proxyInitialMiddleware(ghandlers.ProxyHeaders(commonHandlerPipeline))

	httpHandler := a.httpInitialMiddleware(commonHandlerPipeline)

	// Listen for HTTP
	for _, fd := range a.config.Listeners.HTTP {
		a.listenHTTPFD(&wg, fd, httpHandler, limiter)
	}

	// Listen for HTTPS
	for _, fd := range a.config.Listeners.HTTPS {
		a.listenHTTPSFD(&wg, fd, httpHandler, limiter)
	}

	// Listen for HTTP proxy requests
	for _, fd := range a.config.Listeners.Proxy {
		a.listenProxyFD(&wg, fd, proxyHandler, limiter)
	}

	// Listen for HTTPS PROXYv2 requests
	for _, fd := range a.config.Listeners.HTTPSProxyv2 {
		a.ListenHTTPSProxyv2FD(&wg, fd, httpHandler, limiter)
	}

	// Serve metrics for Prometheus
	if a.config.ListenMetrics != 0 {
		a.listenMetricsFD(&wg, a.config.ListenMetrics)
	}

	a.domains.Read(a.config.General.Domain)

	wg.Wait()
}

func (a *theApp) listenHTTPFD(wg *sync.WaitGroup, fd uintptr, httpHandler http.Handler, limiter *netutil.Limiter) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := a.listenAndServe(listenerConfig{fd: fd, handler: httpHandler, limiter: limiter}); err != nil {
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

		if err := a.listenAndServe(listenerConfig{fd: fd, handler: httpHandler, limiter: limiter, tlsConfig: tlsConfig}); err != nil {
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
			if err := a.listenAndServe(listenerConfig{fd: fd, handler: proxyHandler, limiter: limiter}); err != nil {
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

		if err := a.listenAndServe(listenerConfig{fd: fd, handler: httpHandler, limiter: limiter, tlsConfig: tlsConfig, isProxyV2: true}); err != nil {
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

func runApp(config *cfg.Config) {
	domains, err := source.NewDomains(config.General.DomainConfigurationSource, &config.GitLab)
	if err != nil {
		log.WithError(err).Fatal("could not create domains config source")
	}

	knownPaths := []string{
		pathPrefixArtifacts,
		pathPrefixAuth,
		config.General.StatusPath,
	}

	a := theApp{config: config, domains: domains, knownPaths: knownPaths}

	err = logging.ConfigureLogging(a.config.Log.Format, a.config.Log.Verbose)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize logging")
	}

	if config.ArtifactsServer.URL != "" {
		a.Artifact = artifact.New(config.ArtifactsServer.URL, config.ArtifactsServer.TimeoutSeconds, config.General.Domain)
	}

	a.setAuth(config)

	a.Handlers = handlers.New(a.Auth, a.Artifact)

	// TODO: This if was introduced when `gitlab-server` wasn't a required parameter
	// once we completely remove support for legacy architecture and make it required
	// we can just remove this if statement https://gitlab.com/gitlab-org/gitlab-pages/-/issues/581
	if config.GitLab.PublicServer != "" {
		a.AcmeMiddleware = &acme.Middleware{GitlabURL: config.GitLab.PublicServer}
	}

	if len(config.General.CustomHeaders) != 0 {
		customHeaders, err := middleware.ParseHeaderString(config.General.CustomHeaders)
		if err != nil {
			log.WithError(err).Fatal("Unable to parse header string")
		}
		a.CustomHeaders = customHeaders
	}

	if err := mimedb.LoadTypes(); err != nil {
		log.WithError(err).Warn("Loading extended MIME database failed")
	}

	// TODO: reconfigure all VFS'
	//  https://gitlab.com/gitlab-org/gitlab-pages/-/issues/512
	if err := zip.Instance().Reconfigure(config); err != nil {
		fatal(err, "failed to reconfigure zip VFS")
	}

	a.Run()
}

func (a *theApp) setAuth(config *cfg.Config) {
	if config.Authentication.ClientID == "" {
		return
	}

	var err error
	a.Auth, err = auth.New(config.General.Domain, config.Authentication.Secret, config.Authentication.ClientID, config.Authentication.ClientSecret,
		config.Authentication.RedirectURI, config.GitLab.InternalServer, config.GitLab.PublicServer, config.Authentication.Scope)
	if err != nil {
		log.WithError(err).Fatal("could not initialize auth package")
	}
}

// fatal will log a fatal error and exit.
func fatal(err error, message string) {
	log.WithError(err).Fatal(message)
}

func (a *theApp) TLSConfig() (*cryptotls.Config, error) {
	return tls.Create(a.config.General.RootCertificate, a.config.General.RootKey, a.ServeTLS,
		a.config.General.InsecureCiphers, a.config.TLS.MinVersion, a.config.TLS.MaxVersion)
}
