package config

import (
	"crypto/tls"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/namsral/flag"
)

var (
	// allTLSVersions has all supported flag values
	allTLSVersions = map[string]uint16{
		"":       0, // Default value in tls.Config
		"tls1.2": tls.VersionTLS12,
		"tls1.3": tls.VersionTLS13,
	}

	pagesRootCert = flag.String("root-cert", "", "The default path to file certificate to serve static pages")
	pagesRootKey  = flag.String("root-key", "", "The default path to file certificate to serve static pages")
	redirectHTTP  = flag.Bool("redirect-http", false, "Redirect pages from HTTP to HTTPS")
	_             = flag.Bool("use-http2", true, "DEPRECATED: HTTP2 is always enabled for pages")
	pagesRoot     = flag.String("pages-root", "shared/pages", "The directory where pages are stored")
	pagesDomain   = flag.String("pages-domain", "gitlab-example.com", "The domain to serve static pages")

	// HTTP rate limits
	rateLimitSourceIP      = flag.Float64("rate-limit-source-ip", 0.0, "Rate limit HTTP requests per second from a single IP, 0 means is disabled")
	rateLimitSourceIPBurst = flag.Int("rate-limit-source-ip-burst", 100, "Rate limit HTTP requests from a single IP, maximum burst allowed per second")
	rateLimitDomain        = flag.Float64("rate-limit-domain", 0.0, "Rate limit HTTP requests per second to a single domain, 0 means is disabled")
	rateLimitDomainBurst   = flag.Int("rate-limit-domain-burst", 100, "Rate limit HTTP requests to a single domain, maximum burst allowed per second")
	// TLS connections rate limits
	rateLimitTLSSourceIP      = flag.Float64("rate-limit-tls-source-ip", 0.0, "Rate limit new TLS connections per second from a single IP, 0 means is disabled")
	rateLimitTLSSourceIPBurst = flag.Int("rate-limit-tls-source-ip-burst", 100, "Rate limit new TLS connections from a single IP, maximum burst allowed per second")
	rateLimitTLSDomain        = flag.Float64("rate-limit-tls-domain", 0.0, "Rate limit new TLS connections per second from to a single domain, 0 means is disabled")
	rateLimitTLSDomainBurst   = flag.Int("rate-limit-tls-domain-burst", 100, "Rate limit new TLS connections from a single domain, maximum burst allowed per second")

	artifactsServer         = flag.String("artifacts-server", "", "API URL to proxy artifact requests to, e.g.: 'https://gitlab.com/api/v4'")
	artifactsServerTimeout  = flag.Int("artifacts-server-timeout", 10, "Timeout (in seconds) for a proxied request to the artifacts server")
	pagesStatus             = flag.String("pages-status", "", "The url path for a status page, e.g., /@status")
	metricsAddress          = flag.String("metrics-address", "", "The address to listen on for metrics requests")
	sentryDSN               = flag.String("sentry-dsn", "", "The address for sending sentry crash reporting to")
	sentryEnvironment       = flag.String("sentry-environment", "", "The environment for sentry crash reporting")
	_                       = flag.Uint("daemon-uid", 0, "DEPRECATED and ignored, will be removed in 15.0")
	_                       = flag.Uint("daemon-gid", 0, "DEPRECATED and ignored, will be removed in 15.0")
	_                       = flag.Bool("daemon-enable-jail", false, "DEPRECATED and ignored, will be removed in 15.0")
	_                       = flag.Bool("daemon-inplace-chroot", false, "DEPRECATED and ignored, will be removed in 15.0") // TODO: https://gitlab.com/gitlab-org/gitlab-pages/-/issues/599
	propagateCorrelationID  = flag.Bool("propagate-correlation-id", false, "Reuse existing Correlation-ID from the incoming request header `X-Request-ID` if present")
	serverShutdownTimeout   = flag.Duration("server-shutdown-timeout", 30*time.Second, "GitLab Pages server shutdown timeout (default: 30s)")
	logFormat               = flag.String("log-format", "json", "The log output format: 'text' or 'json'")
	logVerbose              = flag.Bool("log-verbose", false, "Verbose logging")
	secret                  = flag.String("auth-secret", "", "Cookie store hash key, should be at least 32 bytes long")
	publicGitLabServer      = flag.String("gitlab-server", "", "Public GitLab server, for example https://www.gitlab.com")
	internalGitLabServer    = flag.String("internal-gitlab-server", "", "Internal GitLab server used for API requests, useful if you want to send that traffic over an internal load balancer, example value https://gitlab.example.internal (defaults to value of gitlab-server)")
	gitLabAPISecretKey      = flag.String("api-secret-key", "", "File with secret key used to authenticate with the GitLab API")
	gitlabClientHTTPTimeout = flag.Duration("gitlab-client-http-timeout", 10*time.Second, "GitLab API HTTP client connection timeout in seconds (default: 10s)")
	gitlabClientJWTExpiry   = flag.Duration("gitlab-client-jwt-expiry", 30*time.Second, "JWT Token expiry time in seconds (default: 30s)")
	gitlabCacheExpiry       = flag.Duration("gitlab-cache-expiry", 10*time.Minute, "The maximum time a domain's configuration is stored in the cache")
	gitlabCacheRefresh      = flag.Duration("gitlab-cache-refresh", time.Minute, "The interval at which a domain's configuration is set to be due to refresh")
	gitlabCacheCleanup      = flag.Duration("gitlab-cache-cleanup", time.Minute, "The interval at which expired items are removed from the cache")
	gitlabRetrievalTimeout  = flag.Duration("gitlab-retrieval-timeout", 30*time.Second, "The maximum time to wait for a response from the GitLab API per request")
	gitlabRetrievalInterval = flag.Duration("gitlab-retrieval-interval", time.Second, "The interval to wait before retrying to resolve a domain's configuration via the GitLab API")
	gitlabRetrievalRetries  = flag.Int("gitlab-retrieval-retries", 3, "The maximum number of times to retry to resolve a domain's configuration via the API")

	_          = flag.String("domain-config-source", "gitlab", "DEPRECATED and has not affect, see https://gitlab.com/gitlab-org/gitlab-pages/-/merge_requests/541")
	enableDisk = flag.Bool("enable-disk", true, "Enable disk access, shall be disabled in environments where shared disk storage isn't available")

	clientID           = flag.String("auth-client-id", "", "GitLab application Client ID")
	clientSecret       = flag.String("auth-client-secret", "", "GitLab application Client Secret")
	redirectURI        = flag.String("auth-redirect-uri", "", "GitLab application redirect URI")
	authScope          = flag.String("auth-scope", "api", "Scope to be used for authentication (must match GitLab Pages OAuth application settings)")
	maxConns           = flag.Int("max-conns", 0, "Limit on the number of concurrent connections to the HTTP, HTTPS or proxy listeners, 0 for no limit")
	maxURILength       = flag.Int("max-uri-length", 1024, "Limit the length of URI, 0 for unlimited.")
	insecureCiphers    = flag.Bool("insecure-ciphers", false, "Use default list of cipher suites, may contain insecure ones like 3DES and RC4")
	tlsMinVersion      = flag.String("tls-min-version", "tls1.2", tlsVersionFlagUsage("min"))
	tlsMaxVersion      = flag.String("tls-max-version", "", tlsVersionFlagUsage("max"))
	zipCacheExpiration = flag.Duration("zip-cache-expiration", 60*time.Second, "Zip serving archive cache expiration interval")
	zipCacheCleanup    = flag.Duration("zip-cache-cleanup", 30*time.Second, "Zip serving archive cache cleanup interval")
	zipCacheRefresh    = flag.Duration("zip-cache-refresh", 30*time.Second, "Zip serving archive cache refresh interval")
	zipOpenTimeout     = flag.Duration("zip-open-timeout", 30*time.Second, "Zip archive open timeout")

	disableCrossOriginRequests = flag.Bool("disable-cross-origin-requests", false, "Disable cross-origin requests")

	showVersion = flag.Bool("version", false, "Show version")

	// See initFlags()
	listenHTTP         = MultiStringFlag{separator: ","}
	listenHTTPS        = MultiStringFlag{separator: ","}
	listenProxy        = MultiStringFlag{separator: ","}
	listenHTTPSProxyv2 = MultiStringFlag{separator: ","}

	header = MultiStringFlag{separator: ";;"}
)

// initFlags will be called from LoadConfig
func initFlags() {
	flag.Var(&listenHTTP, "listen-http", "The address(es) to listen on for HTTP requests")
	flag.Var(&listenHTTPS, "listen-https", "The address(es) to listen on for HTTPS requests")
	flag.Var(&listenProxy, "listen-proxy", "The address(es) to listen on for proxy requests")
	flag.Var(&listenHTTPSProxyv2, "listen-https-proxyv2", "The address(es) to listen on for HTTPS PROXYv2 requests (https://www.haproxy.org/download/1.8/doc/proxy-protocol.txt)")
	flag.Var(&header, "header", "The additional http header(s) that should be send to the client")

	// read from -config=/path/to/gitlab-pages-config
	flag.String(flag.DefaultConfigFlagname, "", "path to config file")

	flag.Parse()
}

// tlsVersionFlagUsage returns string with explanation how to use the tls version CLI flag
func tlsVersionFlagUsage(minOrMax string) string {
	versions := []string{}

	for version := range allTLSVersions {
		if version != "" {
			versions = append(versions, fmt.Sprintf("%q", version))
		}
	}
	sort.Strings(versions)

	return fmt.Sprintf("Specifies the "+minOrMax+"imum SSL/TLS version, supported values are %s", strings.Join(versions, ", "))
}
