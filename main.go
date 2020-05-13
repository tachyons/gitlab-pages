package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"
	"gitlab.com/gitlab-org/labkit/errortracking"

	"gitlab.com/gitlab-org/gitlab-pages/internal/host"
	"gitlab.com/gitlab-org/gitlab-pages/internal/logging"
	"gitlab.com/gitlab-org/gitlab-pages/internal/tlsconfig"
	"gitlab.com/gitlab-org/gitlab-pages/internal/validateargs"
	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

// VERSION stores the information about the semantic version of application
var VERSION = "dev"

// REVISION stores the information about the git revision of application
var REVISION = "HEAD"

func init() {
	flag.Var(&listenHTTP, "listen-http", "The address(es) to listen on for HTTP requests")
	flag.Var(&listenHTTPS, "listen-https", "The address(es) to listen on for HTTPS requests")
	flag.Var(&listenProxy, "listen-proxy", "The address(es) to listen on for proxy requests")
	flag.Var(&header, "header", "The additional http header(s) that should be send to the client")
}

var (
	pagesRootCert           = flag.String("root-cert", "", "The default path to file certificate to serve static pages")
	pagesRootKey            = flag.String("root-key", "", "The default path to file certificate to serve static pages")
	redirectHTTP            = flag.Bool("redirect-http", false, "Redirect pages from HTTP to HTTPS")
	useHTTP2                = flag.Bool("use-http2", true, "Enable HTTP2 support")
	pagesRoot               = flag.String("pages-root", "shared/pages", "The directory where pages are stored")
	pagesDomain             = flag.String("pages-domain", "gitlab-example.com", "The domain to serve static pages")
	artifactsServer         = flag.String("artifacts-server", "", "API URL to proxy artifact requests to, e.g.: 'https://gitlab.com/api/v4'")
	artifactsServerTimeout  = flag.Int("artifacts-server-timeout", 10, "Timeout (in seconds) for a proxied request to the artifacts server")
	pagesStatus             = flag.String("pages-status", "", "The url path for a status page, e.g., /@status")
	metricsAddress          = flag.String("metrics-address", "", "The address to listen on for metrics requests")
	sentryDSN               = flag.String("sentry-dsn", "", "The address for sending sentry crash reporting to")
	sentryEnvironment       = flag.String("sentry-environment", "", "The environment for sentry crash reporting")
	daemonUID               = flag.Uint("daemon-uid", 0, "Drop privileges to this user")
	daemonGID               = flag.Uint("daemon-gid", 0, "Drop privileges to this group")
	daemonInplaceChroot     = flag.Bool("daemon-inplace-chroot", false, "Fall back to a non-bind-mount chroot of -pages-root when daemonizing")
	logFormat               = flag.String("log-format", "text", "The log output format: 'text' or 'json'")
	logVerbose              = flag.Bool("log-verbose", false, "Verbose logging")
	_                       = flag.String("admin-secret-path", "", "DEPRECATED")
	_                       = flag.String("admin-unix-listener", "", "DEPRECATED")
	_                       = flag.String("admin-https-listener", "", "DEPRECATED")
	_                       = flag.String("admin-https-cert", "", "DEPRECATED")
	_                       = flag.String("admin-https-key", "", "DEPRECATED")
	secret                  = flag.String("auth-secret", "", "Cookie store hash key, should be at least 32 bytes long")
	gitLabAuthServer        = flag.String("auth-server", "", "DEPRECATED, use gitlab-server instead. GitLab server, for example https://www.gitlab.com")
	gitLabServer            = flag.String("gitlab-server", "", "GitLab server, for example https://www.gitlab.com")
	internalGitLabServer    = flag.String("internal-gitlab-server", "", "Internal GitLab server used for API requests, useful if you want to send that traffic over an internal load balancer, example value https://www.gitlab.com (defaults to value of gitlab-server)")
	gitLabAPISecretKey      = flag.String("api-secret-key", "", "File with secret key used to authenticate with the GitLab API")
	gitlabClientHTTPTimeout = flag.Duration("gitlab-client-http-timeout", 10*time.Second, "GitLab API HTTP client connection timeout in seconds (default: 10s)")
	gitlabClientJWTExpiry   = flag.Duration("gitlab-client-jwt-expiry", 30*time.Second, "JWT Token expiry time in seconds (default: 30s)")
	clientID                = flag.String("auth-client-id", "", "GitLab application Client ID")
	clientSecret            = flag.String("auth-client-secret", "", "GitLab application Client Secret")
	redirectURI             = flag.String("auth-redirect-uri", "", "GitLab application redirect URI")
	maxConns                = flag.Uint("max-conns", 5000, "Limit on the number of concurrent connections to the HTTP, HTTPS or proxy listeners")
	insecureCiphers         = flag.Bool("insecure-ciphers", false, "Use default list of cipher suites, may contain insecure ones like 3DES and RC4")
	tlsMinVersion           = flag.String("tls-min-version", "tls1.2", tlsconfig.FlagUsage("min"))
	tlsMaxVersion           = flag.String("tls-max-version", "", tlsconfig.FlagUsage("max"))

	disableCrossOriginRequests = flag.Bool("disable-cross-origin-requests", false, "Disable cross-origin requests")

	// See init()
	listenHTTP  MultiStringFlag
	listenHTTPS MultiStringFlag
	listenProxy MultiStringFlag

	header MultiStringFlag
)

var (
	errArtifactSchemaUnsupported   = errors.New("artifacts-server scheme must be either http:// or https://")
	errArtifactsServerTimeoutValue = errors.New("artifacts-server-timeout must be greater than or equal to 1")

	errSecretNotDefined       = errors.New("auth-secret must be defined if authentication is supported")
	errClientIDNotDefined     = errors.New("auth-client-id must be defined if authentication is supported")
	errClientSecretNotDefined = errors.New("auth-client-secret must be defined if authentication is supported")
	errGitLabServerNotDefined = errors.New("gitlab-server must be defined if authentication is supported")
	errRedirectURINotDefined  = errors.New("auth-redirect-uri must be defined if authentication is supported")
)

func gitlabServerFromFlags() string {
	if *gitLabServer != "" {
		return *gitLabServer
	}

	if *gitLabAuthServer != "" {
		log.Warn("auth-server parameter is deprecated, use gitlab-server instead")
		return *gitLabAuthServer
	}

	url, _ := url.Parse(*artifactsServer)
	return host.FromString(url.Host)
}

func internalGitLabServerFromFlags() string {
	if *internalGitLabServer != "" {
		return *internalGitLabServer
	}

	return gitlabServerFromFlags()
}

func setArtifactsServer(artifactsServer string, artifactsServerTimeout int, config *appConfig) {
	u, err := url.Parse(artifactsServer)
	if err != nil {
		log.Fatal(err)
	}
	// url.Parse ensures that the Scheme arttribute is always lower case.
	if u.Scheme != "http" && u.Scheme != "https" {
		errortracking.Capture(err)
		log.Fatal(errArtifactSchemaUnsupported)
	}

	if artifactsServerTimeout < 1 {
		errortracking.Capture(err)
		log.Fatal(errArtifactsServerTimeoutValue)
	}

	config.ArtifactsServerTimeout = artifactsServerTimeout
	config.ArtifactsServer = artifactsServer
}

func setGitLabAPISecretKey(secretFile string, config *appConfig) {
	encoded := readFile(secretFile)

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	secretLength, err := base64.StdEncoding.Decode(decoded, encoded)
	if err != nil {
		log.WithError(err).Fatal("Failed to decode GitLab API secret")
	}

	if secretLength != 32 {
		log.WithError(fmt.Errorf("Expected 32 bytes GitLab API secret but got %d bytes", secretLength)).Fatal("Failed to decode GitLab API secret")
	}

	config.GitLabAPISecretKey = decoded
}

func configFromFlags() appConfig {
	var config appConfig

	config.Domain = strings.ToLower(*pagesDomain)
	config.RedirectHTTP = *redirectHTTP
	config.HTTP2 = *useHTTP2
	config.DisableCrossOriginRequests = *disableCrossOriginRequests
	config.StatusPath = *pagesStatus
	config.LogFormat = *logFormat
	config.LogVerbose = *logVerbose
	config.MaxConns = int(*maxConns)
	config.InsecureCiphers = *insecureCiphers
	// tlsMinVersion and tlsMaxVersion are validated in appMain
	config.TLSMinVersion = tlsconfig.AllTLSVersions[*tlsMinVersion]
	config.TLSMaxVersion = tlsconfig.AllTLSVersions[*tlsMaxVersion]
	config.CustomHeaders = header

	for _, file := range []struct {
		contents *[]byte
		path     string
	}{
		{&config.RootCertificate, *pagesRootCert},
		{&config.RootKey, *pagesRootKey},
	} {
		if file.path != "" {
			*file.contents = readFile(file.path)
		}
	}

	if *gitLabAPISecretKey != "" {
		setGitLabAPISecretKey(*gitLabAPISecretKey, &config)
	}

	if *artifactsServer != "" {
		setArtifactsServer(*artifactsServer, *artifactsServerTimeout, &config)
	}

	config.GitLabServer = gitlabServerFromFlags()
	config.InternalGitLabServer = internalGitLabServerFromFlags()
	config.GitlabClientHTTPTimeout = *gitlabClientHTTPTimeout
	config.GitlabJWTTokenExpiration = *gitlabClientJWTExpiry
	config.StoreSecret = *secret
	config.ClientID = *clientID
	config.ClientSecret = *clientSecret
	config.RedirectURI = *redirectURI
	config.SentryDSN = *sentryDSN
	config.SentryEnvironment = *sentryEnvironment

	checkAuthenticationConfig(config)

	return config
}

func checkAuthenticationConfig(config appConfig) {
	if config.StoreSecret == "" && config.ClientID == "" &&
		config.ClientSecret == "" && config.RedirectURI == "" {
		return
	}
	assertAuthConfig(config)
}

func assertAuthConfig(config appConfig) {
	if config.StoreSecret == "" {
		log.Fatal(errSecretNotDefined)
	}
	if config.ClientID == "" {
		log.Fatal(errClientIDNotDefined)
	}
	if config.ClientSecret == "" {
		log.Fatal(errClientSecretNotDefined)
	}
	if config.GitLabServer == "" {
		log.Fatal(errGitLabServerNotDefined)
	}
	if config.RedirectURI == "" {
		log.Fatal(errRedirectURINotDefined)
	}
}

func initErrorReporting(sentryDSN, sentryEnvironment string) {
	errortracking.Initialize(
		errortracking.WithSentryDSN(sentryDSN),
		errortracking.WithVersion(fmt.Sprintf("%s-%s", VERSION, REVISION)),
		errortracking.WithLoggerName("gitlab-pages"),
		errortracking.WithSentryEnvironment(sentryEnvironment))
}

func loadConfig() appConfig {
	if err := validateargs.NotAllowed(os.Args[1:]); err != nil {
		log.WithError(err).Fatal("Using invalid arguments, use -config=gitlab-pages-config file instead")
	}

	if err := validateargs.Deprecated(os.Args[1:]); err != nil {
		log.WithError(err).Warn("Using deprecated arguments")
	}

	config := configFromFlags()
	if config.SentryDSN != "" {
		initErrorReporting(config.SentryDSN, config.SentryEnvironment)
	}

	log.WithFields(log.Fields{
		"artifacts-server":              *artifactsServer,
		"artifacts-server-timeout":      *artifactsServerTimeout,
		"daemon-gid":                    *daemonGID,
		"daemon-uid":                    *daemonUID,
		"daemon-inplace-chroot":         *daemonInplaceChroot,
		"default-config-filename":       flag.DefaultConfigFlagname,
		"disable-cross-origin-requests": *disableCrossOriginRequests,
		"domain":                        config.Domain,
		"insecure-ciphers":              config.InsecureCiphers,
		"listen-http":                   strings.Join(listenHTTP, ","),
		"listen-https":                  strings.Join(listenHTTPS, ","),
		"listen-proxy":                  strings.Join(listenProxy, ","),
		"log-format":                    *logFormat,
		"metrics-address":               *metricsAddress,
		"pages-domain":                  *pagesDomain,
		"pages-root":                    *pagesRoot,
		"pages-status":                  *pagesStatus,
		"redirect-http":                 config.RedirectHTTP,
		"root-cert":                     *pagesRootKey,
		"root-key":                      *pagesRootCert,
		"status_path":                   config.StatusPath,
		"tls-min-version":               *tlsMinVersion,
		"tls-max-version":               *tlsMaxVersion,
		"use-http-2":                    config.HTTP2,
		"gitlab-server":                 config.GitLabServer,
		"internal-gitlab-server":        config.InternalGitLabServer,
		"api-secret-key":                *gitLabAPISecretKey,
		"auth-redirect-uri":             config.RedirectURI,
	}).Debug("Start daemon with configuration")

	return config
}

func appMain() {
	var showVersion = flag.Bool("version", false, "Show version")

	// read from -config=/path/to/gitlab-pages-config
	flag.String(flag.DefaultConfigFlagname, "", "path to config file")
	flag.Parse()
	if err := tlsconfig.ValidateTLSVersions(*tlsMinVersion, *tlsMaxVersion); err != nil {
		fatal(err, "invalid TLS version")
	}

	printVersion(*showVersion, VERSION)

	err := logging.ConfigureLogging(*logFormat, *logVerbose)
	if err != nil {
		log.WithError(err).Fatal("Failed to initialize logging")
	}

	log.WithFields(log.Fields{
		"version":  VERSION,
		"revision": REVISION,
	}).Print("GitLab Pages Daemon")
	log.Printf("URL: https://gitlab.com/gitlab-org/gitlab-pages")

	if err := os.Chdir(*pagesRoot); err != nil {
		fatal(err, "could not change directory into pagesRoot")
	}

	config := loadConfig()

	for _, cs := range [][]io.Closer{
		createAppListeners(&config),
		createMetricsListener(&config),
	} {
		defer closeAll(cs)
	}

	if *daemonUID != 0 || *daemonGID != 0 {
		if err := daemonize(config, *daemonUID, *daemonGID, *daemonInplaceChroot); err != nil {
			errortracking.Capture(err)
			fatal(err, "could not create pages daemon")
		}

		return
	}

	runApp(config)
}

func closeAll(cs []io.Closer) {
	for _, c := range cs {
		c.Close()
	}
}

// createAppListeners returns net.Listener and *os.File instances. The
// caller must ensure they don't get closed or garbage-collected (which
// implies closing) too soon.
func createAppListeners(config *appConfig) []io.Closer {
	var closers []io.Closer

	for _, addr := range listenHTTP.Split() {
		l, f := createSocket(addr)
		closers = append(closers, l, f)

		log.WithFields(log.Fields{
			"listener": addr,
		}).Debug("Set up HTTP listener")

		config.ListenHTTP = append(config.ListenHTTP, f.Fd())
	}

	for _, addr := range listenHTTPS.Split() {
		l, f := createSocket(addr)
		closers = append(closers, l, f)

		log.WithFields(log.Fields{
			"listener": addr,
		}).Debug("Set up HTTPS listener")

		config.ListenHTTPS = append(config.ListenHTTPS, f.Fd())
	}

	for _, addr := range listenProxy.Split() {
		l, f := createSocket(addr)
		closers = append(closers, l, f)

		log.WithFields(log.Fields{
			"listener": addr,
		}).Debug("Set up proxy listener")

		config.ListenProxy = append(config.ListenProxy, f.Fd())
	}

	return closers
}

// createMetricsListener returns net.Listener and *os.File instances. The
// caller must ensure they don't get closed or garbage-collected (which
// implies closing) too soon.
func createMetricsListener(config *appConfig) []io.Closer {
	addr := *metricsAddress
	if addr == "" {
		return nil
	}

	l, f := createSocket(addr)
	config.ListenMetrics = f.Fd()

	log.WithFields(log.Fields{
		"listener": addr,
	}).Debug("Set up metrics listener")

	return []io.Closer{l, f}
}

func printVersion(showVersion bool, version string) {
	if showVersion {
		fmt.Fprintf(os.Stdout, "%s\n", version)
		os.Exit(0)
	}
}

func main() {
	log.SetOutput(os.Stderr)

	rand.Seed(time.Now().UnixNano())

	metrics.MustRegister()

	daemonMain()
	appMain()
}
