package main

import (
	"context"
	cryptotls "crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	ghandlers "github.com/gorilla/handlers"
	"github.com/rs/cors"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/go-mimedb"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/errortracking"
	labmetrics "gitlab.com/gitlab-org/labkit/metrics"
	"gitlab.com/gitlab-org/labkit/monitoring"

	"gitlab.com/gitlab-org/gitlab-pages/internal/acme"
	"gitlab.com/gitlab-org/gitlab-pages/internal/artifact"
	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	cfg "gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
	"gitlab.com/gitlab-org/gitlab-pages/internal/customheaders"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/handlers"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/netutil"
	"gitlab.com/gitlab-org/gitlab-pages/internal/rejectmethods"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/routing"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/zip"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
	"gitlab.com/gitlab-org/gitlab-pages/internal/urilimiter"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

const (
	xForwardedHost = "X-Forwarded-Host"
)

var (
	corsHandler = cors.New(cors.Options{AllowedMethods: []string{http.MethodGet, http.MethodHead}})
)

type theApp struct {
	config         *cfg.Config
	source         source.Source
	Artifact       *artifact.Artifact
	Auth           *auth.Auth
	Handlers       *handlers.Handlers
	AcmeMiddleware *acme.Middleware
	CustomHeaders  http.Header
}

func (a *theApp) isReady() bool {
	return true
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

func (a *theApp) redirectToHTTPS(w http.ResponseWriter, r *http.Request, statusCode int) {
	u := *r.URL
	u.Scheme = request.SchemeHTTPS
	u.Host = r.Host
	u.User = nil

	http.Redirect(w, r, u.String(), statusCode)
}

func (a *theApp) domain(ctx context.Context, host string) (*domain.Domain, error) {
	return a.source.GetDomain(ctx, host)
}

// checkAuthAndServeNotFound performs the auth process if domain can't be found
// the main purpose of this process is to avoid leaking the project existence/not-existence
// by behaving the same if user has no access to the project or if project simply does not exists
func (a *theApp) checkAuthAndServeNotFound(domain *domain.Domain, w http.ResponseWriter, r *http.Request) {
	// To avoid user knowing if pages exist, we will force user to login and authorize pages
	if a.Auth.CheckAuthenticationWithoutProject(w, r, domain) {
		return
	}

	// auth succeeded try to serve the correct 404 page
	domain.ServeNotFoundAuthFailed(w, r)
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
			errortracking.Capture(err, errortracking.WithStackTrace())
			httperrors.Serve500(w)
			return true
		}

		// redirect to auth and serve not found
		a.checkAuthAndServeNotFound(domain, w, r)
		return true
	}

	if !https && domain.IsHTTPSOnly(r) {
		a.redirectToHTTPS(w, r, http.StatusMovedPermanently)
		return true
	}

	return false
}

// healthCheckMiddleware is serving the application status check
func (a *theApp) healthCheckMiddleware(handler http.Handler) (http.Handler, error) {
	healthCheck := http.HandlerFunc(func(w http.ResponseWriter, _r *http.Request) {
		if a.isReady() {
			w.Write([]byte("success\n"))
		} else {
			http.Error(w, "not yet ready", http.StatusServiceUnavailable)
		}
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

// auxiliaryMiddleware will handle status updates, not-ready requests and other
// not static-content responses
func (a *theApp) auxiliaryMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host := domain.GetHost(r)
		domain := domain.FromRequest(r)
		https := request.IsHTTPS(r)

		if a.tryAuxiliaryHandlers(w, r, https, host, domain) {
			return
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

		domain := domain.FromRequest(r)
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

// TODO: move the pipeline configuration to internal/pipeline https://gitlab.com/gitlab-org/gitlab-pages/-/issues/670
func (a *theApp) buildHandlerPipeline() (http.Handler, error) {
	// Handlers should be applied in a reverse order
	handler := a.serveFileOrNotFoundHandler()
	if !a.config.General.DisableCrossOriginRequests {
		handler = corsHandler.Handler(handler)
	}
	handler = a.Auth.AuthorizationMiddleware(handler)
	handler = a.auxiliaryMiddleware(handler)
	handler = a.Auth.AuthenticationMiddleware(handler, a.source)
	handler = a.AcmeMiddleware.AcmeMiddleware(handler)
	handler, err := logging.BasicAccessLogger(handler, a.config.Log.Format, domain.LogFields)
	if err != nil {
		return nil, err
	}

	// Metrics
	metricsMiddleware := labmetrics.NewHandlerFactory(labmetrics.WithNamespace("gitlab_pages"))
	handler = metricsMiddleware(handler)

	handler = routing.NewMiddleware(handler, a.source)

	handler = handlers.Ratelimiter(handler, a.config)

	// Health Check
	handler, err = a.healthCheckMiddleware(handler)
	if err != nil {
		return nil, err
	}

	// Custom response headers
	handler = customheaders.NewMiddleware(handler, a.CustomHeaders)

	// Correlation ID injection middleware
	var correlationOpts []correlation.InboundHandlerOption
	if a.config.General.PropagateCorrelationID {
		correlationOpts = append(correlationOpts, correlation.WithPropagation())
	}
	handler = handlePanicMiddleware(handler)
	handler = correlation.InjectCorrelationID(handler, correlationOpts...)

	// These middlewares MUST be added in the end.
	// Being last means they will be evaluated first
	// preventing any operation on bogus requests.
	handler = urilimiter.NewMiddleware(handler, a.config.General.MaxURILength)
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
	source, err := gitlab.New(&config.GitLab)
	if err != nil {
		log.WithError(err).Fatal("could not create domains config source")
	}

	a := theApp{config: config, source: source}

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
		customHeaders, err := customheaders.ParseHeaderString(config.General.CustomHeaders)
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

// handlePanicMiddleware logs and captures the recover() information from any panic
func handlePanicMiddleware(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			i := recover()
			if i != nil {
				err := fmt.Errorf("panic trace: %v", i)
				metrics.PanicRecoveredCount.Inc()
				logging.LogRequest(r).WithError(err).Error("recovered from panic")
				errortracking.Capture(err, errortracking.WithRequest(r), errortracking.WithContext(r.Context()), errortracking.WithStackTrace())
				httperrors.Serve500(w)
			}
		}()

		handler.ServeHTTP(w, r)
	})
}
