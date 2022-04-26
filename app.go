package main

import (
	"context"
	cryptotls "crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	ghandlers "github.com/gorilla/handlers"
	"github.com/hashicorp/go-multierror"
	"github.com/rs/cors"
	"gitlab.com/gitlab-org/go-mimedb"
	"gitlab.com/gitlab-org/labkit/correlation"
	"gitlab.com/gitlab-org/labkit/log"
	labmetrics "gitlab.com/gitlab-org/labkit/metrics"
	"gitlab.com/gitlab-org/labkit/monitoring"

	"gitlab.com/gitlab-org/gitlab-pages/internal/acme"
	"gitlab.com/gitlab-org/gitlab-pages/internal/artifact"
	"gitlab.com/gitlab-org/gitlab-pages/internal/auth"
	cfg "gitlab.com/gitlab-org/gitlab-pages/internal/config"
	"gitlab.com/gitlab-org/gitlab-pages/internal/customheaders"
	"gitlab.com/gitlab-org/gitlab-pages/internal/domain"
	"gitlab.com/gitlab-org/gitlab-pages/internal/errortracking"
	"gitlab.com/gitlab-org/gitlab-pages/internal/handlers"
	health "gitlab.com/gitlab-org/gitlab-pages/internal/healthcheck"
	"gitlab.com/gitlab-org/gitlab-pages/internal/httperrors"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/netutil"
	"gitlab.com/gitlab-org/gitlab-pages/internal/rejectmethods"
	"gitlab.com/gitlab-org/gitlab-pages/internal/request"
	"gitlab.com/gitlab-org/gitlab-pages/internal/routing"
	"gitlab.com/gitlab-org/gitlab-pages/internal/serving/disk/zip"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source"
	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab"
	"gitlab.com/gitlab-org/gitlab-pages/internal/tls"
	"gitlab.com/gitlab-org/gitlab-pages/internal/urilimiter"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

var (
	corsHandler = cors.New(cors.Options{AllowedMethods: []string{http.MethodGet, http.MethodHead}})
)

type theApp struct {
	config         *cfg.Config
	source         source.Source
	tlsConfig      *cryptotls.Config
	Artifact       *artifact.Artifact
	Auth           *auth.Auth
	Handlers       *handlers.Handlers
	AcmeMiddleware *acme.Middleware
	CustomHeaders  http.Header
}

func (a *theApp) GetCertificate(ch *cryptotls.ClientHelloInfo) (*cryptotls.Certificate, error) {
	if ch.ServerName == "" {
		return nil, nil
	}

	if domain, _ := a.domain(context.Background(), ch.ServerName); domain != nil {
		certificate, _ := domain.EnsureCertificate()
		return certificate, nil
	}

	return nil, nil
}

func (a *theApp) getTLSConfig() (*cryptotls.Config, error) {
	// we call this function only when tls config is needed, and we ignore TLS related flags otherwise
	// in theory you can configure both listen-https and listen-proxyv2,
	// so this return is here to have a single TLS config
	if a.tlsConfig != nil {
		return a.tlsConfig, nil
	}

	var err error
	a.tlsConfig, err = tls.GetTLSConfig(a.config, a.GetCertificate)

	return a.tlsConfig, err
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
	if a.Handlers.HandleArtifactRequest(host, w, r) {
		return true
	}

	if _, err := domain.GetLookupPath(r); err != nil {
		if errors.Is(err, gitlab.ErrDiskDisabled) {
			errortracking.CaptureErrWithReqAndStackTrace(err, r)
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

	handler = routing.NewMiddleware(handler, a.source)

	// Add auto redirect
	handler = handlers.HTTPSRedirectMiddleware(handler, a.config.General.RedirectHTTP)

	handler = handlers.Ratelimiter(handler, &a.config.RateLimit)

	// Health Check
	handler = health.NewMiddleware(handler, a.config.General.StatusPath)

	// Custom response headers
	handler = customheaders.NewMiddleware(handler, a.CustomHeaders)

	// Correlation ID injection middleware
	var correlationOpts []correlation.InboundHandlerOption
	if a.config.General.PropagateCorrelationID {
		correlationOpts = append(correlationOpts, correlation.WithPropagation())
	}
	handler = handlePanicMiddleware(handler)

	// Access logs and metrics
	handler, err := logging.BasicAccessLogger(handler, a.config.Log.Format)
	if err != nil {
		return nil, err
	}
	metricsMiddleware := labmetrics.NewHandlerFactory(labmetrics.WithNamespace("gitlab_pages"))
	handler = metricsMiddleware(handler)

	handler = correlation.InjectCorrelationID(handler, correlationOpts...)

	// These middlewares MUST be added in the end.
	// Being last means they will be evaluated first
	// preventing any operation on bogus requests.
	handler = urilimiter.NewMiddleware(handler, a.config.General.MaxURILength)
	handler = rejectmethods.NewMiddleware(handler)

	return handler, nil
}

// nolint: gocyclo // ignore this
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

	proxyHandler := ghandlers.ProxyHeaders(commonHandlerPipeline)

	httpHandler := a.httpInitialMiddleware(commonHandlerPipeline)

	var servers []*http.Server

	// Listen for HTTP
	for _, addr := range a.config.ListenHTTPStrings.Split() {
		s := a.listen(
			addr,
			httpHandler,
			errortracking.WithField("listener", request.SchemeHTTP),
			withLimiter(limiter),
		)
		servers = append(servers, s)
	}

	// Listen for HTTPS
	for _, addr := range a.config.ListenHTTPSStrings.Split() {
		tlsConfig, err := a.getTLSConfig()
		if err != nil {
			log.WithError(err).Fatal("Unable to retrieve tls config")
		}

		s := a.listen(
			addr,
			httpHandler,
			errortracking.WithField("listener", request.SchemeHTTPS),
			withLimiter(limiter),
			withTLSConfig(tlsConfig),
		)
		servers = append(servers, s)
	}

	// Listen for HTTP proxy requests
	for _, addr := range a.config.ListenProxyStrings.Split() {
		s := a.listen(
			addr,
			proxyHandler,
			errortracking.WithField("listener", "http proxy"),
			withLimiter(limiter),
		)
		servers = append(servers, s)
	}

	// Listen for HTTPS PROXYv2 requests
	for _, addr := range a.config.ListenHTTPSProxyv2Strings.Split() {
		tlsConfig, err := a.getTLSConfig()
		if err != nil {
			log.WithError(err).Fatal("Unable to retrieve tls config")
		}

		s := a.listen(
			addr,
			httpHandler,
			errortracking.WithField("listener", "https proxy"),
			withLimiter(limiter),
			withTLSConfig(tlsConfig),
			withProxyV2(),
		)
		servers = append(servers, s)
	}

	// Serve metrics for Prometheus
	if a.config.General.MetricsAddress != "" {
		a.listenMetrics(&wg, a.config.General.MetricsAddress)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)

	<-sigChan

	var result *multierror.Error

	for _, srv := range servers {
		ctx, cancel := context.WithTimeout(context.Background(), a.config.General.ServerShutdownTimeout)

		if err := srv.Shutdown(ctx); err != nil {
			result = multierror.Append(result, err)
		}

		cancel()
	}

	if result.ErrorOrNil() != nil {
		capturingFatal(result)
	}
}

func (a *theApp) listen(addr string, h http.Handler, errTrackingOpt errortracking.CaptureOption, opts ...option) *http.Server {
	server := &http.Server{}
	go func() {
		if err := a.listenAndServe(server, addr, h, opts...); err != nil && !errors.Is(err, http.ErrServerClosed) {
			capturingFatal(err, errTrackingOpt)
		}
	}()

	return server
}

func (a *theApp) listenMetrics(wg *sync.WaitGroup, addr string) {
	wg.Add(1)
	go func() {
		defer wg.Done()

		l, err := net.Listen("tcp", addr)
		if err != nil {
			capturingFatal(fmt.Errorf("failed to listen on addr %s: %w", addr, err), errortracking.WithField("listener", "metrics"))
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
		config.Authentication.RedirectURI, config.GitLab.InternalServer, config.GitLab.PublicServer, config.Authentication.Scope, config.Authentication.Timeout)
	if err != nil {
		log.WithError(err).Fatal("could not initialize auth package")
	}
}

// fatal will log a fatal error and exit.
func fatal(err error, message string) {
	log.WithError(err).Fatal(message)
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
				errortracking.CaptureErrWithReqAndStackTrace(err, r)
				httperrors.Serve500(w)
			}
		}()

		handler.ServeHTTP(w, r)
	})
}
