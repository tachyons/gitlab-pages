package main

import (
	"context"
	cryptotls "crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os/signal"
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
	"golang.org/x/sync/errgroup"

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
	app_router "gitlab.com/gitlab-org/gitlab-pages/internal/router"
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
	config        *cfg.Config
	source        source.Source
	tlsConfig     *cryptotls.Config
	Artifact      *artifact.Artifact
	Auth          *auth.Auth
	Handlers      *handlers.Handlers
	CustomHeaders http.Header
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

func (a *theApp) domain(ctx context.Context, host string) (*domain.Domain, error) {
	return a.source.GetDomain(ctx, host)
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
	handler = handlers.ArtifactMiddleware(handler, a.Handlers)
	handler = a.Auth.AuthenticationMiddleware(handler, a.source)
	handler = handlers.AcmeMiddleware(handler, a.source, a.config.GitLab.PublicServer)

	handler = routing.NewMiddleware(handler, a.source)

	// Add auto redirect
	handler = handlers.HTTPSRedirectMiddleware(handler, a.config.General.RedirectHTTP)

	handler = handlers.Ratelimiter(handler, &a.config.RateLimit)

	metricsMiddleware := labmetrics.NewHandlerFactory(labmetrics.WithNamespace("gitlab_pages"))
	var correlationOpts []correlation.InboundHandlerOption
	if a.config.General.PropagateCorrelationID {
		correlationOpts = append(correlationOpts, correlation.WithPropagation())
	}

	router := app_router.NewRouter(
		rejectmethods.NewMiddleware,
		func(next http.Handler) http.Handler {
			return urilimiter.NewMiddleware(next, a.config.General.MaxURILength)
		},
		func(next http.Handler) http.Handler {
			return correlation.InjectCorrelationID(next, correlationOpts...)
		},
		func(next http.Handler) http.Handler {
			return metricsMiddleware(next)
		},
		func(next http.Handler) http.Handler {
			logHandler, err := logging.BasicAccessLogger(next, a.config.Log.Format)
			if err != nil {
				return next
			}

			return logHandler
		},
		handlePanicMiddleware,
		func(next http.Handler) http.Handler {
			return customheaders.NewMiddleware(next, a.CustomHeaders)
		},
	)

	if a.config.General.StatusPath != "" {
		router.Handle(a.config.General.StatusPath, health.Handler())
	}
	router.Handle("/", handler)

	return router, nil
}

// nolint: gocyclo // ignore this
func (a *theApp) Run() error {
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
		return fmt.Errorf("unable to configure pipeline: %w", err)
	}

	proxyHandler := ghandlers.ProxyHeaders(commonHandlerPipeline)

	httpHandler := a.httpInitialMiddleware(commonHandlerPipeline)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	eg, ctx := errgroup.WithContext(ctx)
	var servers []*http.Server

	// Listen for HTTP
	for _, addr := range a.config.ListenHTTPStrings.Split() {
		s := a.listen(
			eg,
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
			return fmt.Errorf("unable to retrieve tls config: %w", err)
		}

		s := a.listen(
			eg,
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
			eg,
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
			return fmt.Errorf("unable to retrieve tls config: %w", err)
		}

		s := a.listen(
			eg,
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
	if a.config.Metrics.Address != "" {
		s := a.listenMetrics(eg, a.config.Metrics)
		servers = append(servers, s)
	}

	<-ctx.Done()

	var result *multierror.Error

	for _, srv := range servers {
		ctx, cancel := context.WithTimeout(context.Background(), a.config.General.ServerShutdownTimeout)

		if err := srv.Shutdown(ctx); err != nil {
			result = multierror.Append(result, err)
		}

		cancel()
	}

	if err := eg.Wait(); err != nil {
		result = multierror.Append(result, err)
	}

	if result.ErrorOrNil() != nil {
		errortracking.CaptureErrWithStackTrace(result.ErrorOrNil())
		return result.ErrorOrNil()
	}

	return nil
}

func (a *theApp) listen(eg *errgroup.Group, addr string, h http.Handler, errTrackingOpt errortracking.CaptureOption, opts ...option) *http.Server {
	server := &http.Server{}
	eg.Go(func() error {
		if err := a.listenAndServe(server, addr, h, opts...); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errortracking.CaptureErrWithStackTrace(err, errTrackingOpt)
			return err
		}

		return nil
	})

	return server
}

func (a *theApp) listenMetrics(eg *errgroup.Group, config cfg.Metrics) *http.Server {
	server := &http.Server{}
	eg.Go(func() error {
		l, err := net.Listen("tcp", config.Address)
		if err != nil {
			errortracking.CaptureErrWithStackTrace(err, errortracking.WithField("listener", "metrics"))
			return fmt.Errorf("failed to listen on addr %s: %w", config.Address, err)
		}

		if config.TLSConfig != nil {
			l = cryptotls.NewListener(l, config.TLSConfig)
		}

		monitoringOpts := []monitoring.Option{
			monitoring.WithBuildInformation(VERSION, ""),
			monitoring.WithListener(l),
			monitoring.WithServer(server),
		}

		err = monitoring.Start(monitoringOpts...)
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			errortracking.CaptureErrWithStackTrace(err, errortracking.WithField("listener", "metrics"))
			return err
		}

		return nil
	})

	return server
}

func runApp(config *cfg.Config) error {
	source, err := gitlab.New(&config.GitLab)
	if err != nil {
		return fmt.Errorf("could not create domains config source: %w", err)
	}

	a := theApp{config: config, source: source}

	err = logging.ConfigureLogging(a.config.Log.Format, a.config.Log.Verbose)
	if err != nil {
		return fmt.Errorf("failed to initialize logging: %w", err)
	}

	a.Artifact = artifact.New(config.ArtifactsServer.URL, config.ArtifactsServer.TimeoutSeconds, config.General.Domain)

	if err := a.setAuth(config); err != nil {
		return err
	}

	a.Handlers = handlers.New(a.Auth, a.Artifact)

	if len(config.General.CustomHeaders) != 0 {
		customHeaders, err := customheaders.ParseHeaderString(config.General.CustomHeaders)
		if err != nil {
			return fmt.Errorf("unable to parse header string: %w", err)
		}
		a.CustomHeaders = customHeaders
	}

	if err := mimedb.LoadTypes(); err != nil {
		log.WithError(err).Warn("Loading extended MIME database failed")
	}

	// TODO: reconfigure all VFS'
	//  https://gitlab.com/gitlab-org/gitlab-pages/-/issues/512
	if err := zip.Instance().Reconfigure(config); err != nil {
		return fmt.Errorf("failed to reconfigure zip VFS: %w", err)
	}

	return a.Run()
}

func (a *theApp) setAuth(config *cfg.Config) error {
	if config.Authentication.ClientID == "" {
		return nil
	}

	var err error
	a.Auth, err = auth.New(config.General.Domain, config.Authentication.Secret, config.Authentication.ClientID, config.Authentication.ClientSecret,
		config.Authentication.RedirectURI, config.GitLab.InternalServer, config.GitLab.PublicServer, config.Authentication.Scope, config.Authentication.Timeout)
	if err != nil {
		return fmt.Errorf("could not initialize auth package: %w", err)
	}

	return nil
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
