package config

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/namsral/flag"
	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
)

// Config stores all the config options relevant to GitLab Pages.
type Config struct {
	General         General
	ArtifactsServer ArtifactsServer
	Authentication  Auth
	Daemon          Daemon
	GitLab          GitLab
	Listeners       Listeners
	Log             Log
	Sentry          Sentry
	TLS             TLS
	Zip             ZipServing

	// Fields used to share information between files. These are not directly
	// set by command line flags, but rather populated based on info from them.
	// ListenMetrics points to a file descriptor of a socket, whose address is
	// specified by `Config.General.MetricsAddress`.
	ListenMetrics uintptr

	// These fields contain the raw strings passed for listen-http,
	// listen-https, listen-proxy and listen-https-proxyv2 settings. It is used
	// by appmain() to create listeners, and the pointers to these listeners
	// gets assigned to Config.Listeners.* fields
	ListenHTTPStrings         MultiStringFlag
	ListenHTTPSStrings        MultiStringFlag
	ListenProxyStrings        MultiStringFlag
	ListenHTTPSProxyv2Strings MultiStringFlag
}

// General groups settings that are general to GitLab Pages and can not
// be categorized under other head.
type General struct {
	Domain                    string
	DomainConfigurationSource string
	UseLegacyStorage          bool
	HTTP2                     bool
	MaxConns                  int
	MetricsAddress            string
	RedirectHTTP              bool
	RootCertificate           []byte
	RootDir                   string
	RootKey                   []byte
	StatusPath                string

	DisableCrossOriginRequests bool
	InsecureCiphers            bool
	PropagateCorrelationID     bool

	ShowVersion bool

	CustomHeaders []string
}

// ArtifactsServer groups settings related to configuring Artifacts
// server
type ArtifactsServer struct {
	URL            string
	TimeoutSeconds int
}

// Auth groups settings related to configuring Authentication with
// GitLab
type Auth struct {
	Secret       string
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scope        string
}

// Daemon groups settings related to configuring GitLab Pages daemon
type Daemon struct {
	UID           uint
	GID           uint
	InplaceChroot bool
}

// GitLab groups settings related to configuring GitLab client used to
// interact with GitLab API
type GitLab struct {
	Server             string
	InternalServer     string
	APISecretKey       []byte
	ClientHTTPTimeout  time.Duration
	JWTTokenExpiration time.Duration
}

// Listeners groups settings related to configuring various listeners
// (HTTP, HTTPS, Proxy, HTTPSProxyv2)
type Listeners struct {
	HTTP         []uintptr
	HTTPS        []uintptr
	Proxy        []uintptr
	HTTPSProxyv2 []uintptr
}

// Log groups settings related to configuring logging
type Log struct {
	Format  string
	Verbose bool
}

// Sentry groups settings related to configuring Sentry
type Sentry struct {
	DSN         string
	Environment string
}

// TLS groups settings related to configuring TLS
type TLS struct {
	MinVersion uint16
	MaxVersion uint16
}

// ZipServing groups settings to be used by the zip VFS opening and caching
type ZipServing struct {
	ExpirationInterval time.Duration
	CleanupInterval    time.Duration
	RefreshInterval    time.Duration
	OpenTimeout        time.Duration
	AllowedPaths       []string
}

func gitlabServerFromFlags() string {
	if *gitLabServer != "" {
		return *gitLabServer
	}

	if *gitLabAuthServer != "" {
		log.Warn("auth-server parameter is deprecated, use gitlab-server instead")
		return *gitLabAuthServer
	}

	u, err := url.Parse(*artifactsServer)
	if err != nil {
		return ""
	}

	u.Path = ""
	return u.String()
}

func internalGitlabServerFromFlags() string {
	if *internalGitLabServer != "" {
		return *internalGitLabServer
	}

	return gitlabServerFromFlags()
}

func setGitLabAPISecretKey(secretFile string, config *Config) {
	encoded := readFile(secretFile)

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	secretLength, err := base64.StdEncoding.Decode(decoded, encoded)
	if err != nil {
		log.WithError(err).Fatal("Failed to decode GitLab API secret")
	}

	if secretLength != 32 {
		log.WithError(fmt.Errorf("expected 32 bytes GitLab API secret but got %d bytes", secretLength)).Fatal("Failed to decode GitLab API secret")
	}

	config.GitLab.APISecretKey = decoded
}

func checkAuthenticationConfig(config *Config) {
	if config.Authentication.Secret == "" && config.Authentication.ClientID == "" &&
		config.Authentication.ClientSecret == "" && config.Authentication.RedirectURI == "" {
		return
	}
	assertAuthConfig(config)
}

func assertAuthConfig(config *Config) {
	if config.Authentication.Secret == "" {
		log.Fatal("auth-secret must be defined if authentication is supported")
	}
	if config.Authentication.ClientID == "" {
		log.Fatal("auth-client-id must be defined if authentication is supported")
	}
	if config.Authentication.ClientSecret == "" {
		log.Fatal("auth-client-secret must be defined if authentication is supported")
	}
	if config.GitLab.Server == "" {
		log.Fatal("gitlab-server must be defined if authentication is supported")
	}
	if config.Authentication.RedirectURI == "" {
		log.Fatal("auth-redirect-uri must be defined if authentication is supported")
	}
}

func validateArtifactsServer(artifactsServer string, artifactsServerTimeoutSeconds int) {
	u, err := url.Parse(artifactsServer)
	if err != nil {
		log.Fatal(err)
	}
	// url.Parse ensures that the Scheme attribute is always lower case.
	if u.Scheme != "http" && u.Scheme != "https" {
		log.Fatal("artifacts-server scheme must be either http:// or https://")
	}

	if artifactsServerTimeoutSeconds < 1 {
		log.Fatal("artifacts-server-timeout must be greater than or equal to 1")
	}
}

// fatal will log a fatal error and exit.
func fatal(err error, message string) {
	log.WithError(err).Fatal(message)
}

func readFile(file string) (result []byte) {
	result, err := ioutil.ReadFile(file)
	if err != nil {
		fatal(err, "could not read file")
	}
	return
}

// InternalGitLabServerURL returns URL to a GitLab instance.
func (config Config) InternalGitLabServerURL() string {
	return config.GitLab.InternalServer
}

// GitlabClientSecret returns GitLab server access token.
func (config Config) GitlabAPISecret() []byte {
	return config.GitLab.APISecretKey
}

func (config Config) GitlabClientConnectionTimeout() time.Duration {
	return config.GitLab.ClientHTTPTimeout
}

func (config Config) GitlabJWTTokenExpiry() time.Duration {
	return config.GitLab.JWTTokenExpiration
}

func (config Config) DomainConfigSource() string {
	if config.General.UseLegacyStorage {
		return "disk"
	}

	return config.General.DomainConfigurationSource
}

func loadConfig() *Config {
	config := &Config{
		General: General{
			Domain:                     strings.ToLower(*pagesDomain),
			DomainConfigurationSource:  *domainConfigSource,
			UseLegacyStorage:           *useLegacyStorage,
			HTTP2:                      *useHTTP2,
			MaxConns:                   *maxConns,
			MetricsAddress:             *metricsAddress,
			RedirectHTTP:               *redirectHTTP,
			RootDir:                    *pagesRoot,
			StatusPath:                 *pagesStatus,
			DisableCrossOriginRequests: *disableCrossOriginRequests,
			InsecureCiphers:            *insecureCiphers,
			PropagateCorrelationID:     *propagateCorrelationID,
			CustomHeaders:              header.Split(),
			ShowVersion:                *showVersion,
		},
		GitLab: GitLab{
			ClientHTTPTimeout:  *gitlabClientHTTPTimeout,
			JWTTokenExpiration: *gitlabClientJWTExpiry,
		},
		ArtifactsServer: ArtifactsServer{
			TimeoutSeconds: *artifactsServerTimeout,
			URL:            *artifactsServer,
		},
		Authentication: Auth{
			Secret:       *secret,
			ClientID:     *clientID,
			ClientSecret: *clientSecret,
			RedirectURI:  *redirectURI,
			Scope:        *authScope,
		},
		Daemon: Daemon{
			UID:           *daemonUID,
			GID:           *daemonGID,
			InplaceChroot: *daemonInplaceChroot,
		},
		Log: Log{
			Format:  *logFormat,
			Verbose: *logVerbose,
		},
		Sentry: Sentry{
			DSN:         *sentryDSN,
			Environment: *sentryEnvironment,
		},
		TLS: TLS{
			MinVersion: tls.AllTLSVersions[*tlsMinVersion],
			MaxVersion: tls.AllTLSVersions[*tlsMaxVersion],
		},
		Zip: ZipServing{
			ExpirationInterval: *zipCacheExpiration,
			CleanupInterval:    *zipCacheCleanup,
			RefreshInterval:    *zipCacheRefresh,
			OpenTimeout:        *zipOpenTimeout,
			AllowedPaths:       []string{*pagesRoot},
		},

		// Actual listener pointers will be populated in appMain. We populate the
		// raw strings here so that they are available in appMain
		ListenHTTPStrings:         listenHTTP,
		ListenHTTPSStrings:        listenHTTPS,
		ListenProxyStrings:        listenProxy,
		ListenHTTPSProxyv2Strings: listenHTTPSProxyv2,
		Listeners:                 Listeners{},
	}

	// Populating remaining General settings
	for _, file := range []struct {
		contents *[]byte
		path     string
	}{
		{&config.General.RootCertificate, *pagesRootCert},
		{&config.General.RootKey, *pagesRootKey},
	} {
		if file.path != "" {
			*file.contents = readFile(file.path)
		}
	}

	// Populating remaining GitLab settings
	config.GitLab.Server = gitlabServerFromFlags()
	config.GitLab.InternalServer = internalGitlabServerFromFlags()
	if *gitLabAPISecretKey != "" {
		setGitLabAPISecretKey(*gitLabAPISecretKey, config)
	}

	// Validating Artifacts server settings
	if *artifactsServer != "" {
		validateArtifactsServer(*artifactsServer, *artifactsServerTimeout)
	}

	// Validating Authentication settings
	checkAuthenticationConfig(config)

	// Validating TLS settings
	if err := tls.ValidateTLSVersions(*tlsMinVersion, *tlsMaxVersion); err != nil {
		fatal(err, "invalid TLS version")
	}

	return config
}

func LogConfig(config *Config) {
	log.WithFields(log.Fields{
		"artifacts-server":              *artifactsServer,
		"artifacts-server-timeout":      *artifactsServerTimeout,
		"daemon-gid":                    *daemonGID,
		"daemon-uid":                    *daemonUID,
		"daemon-inplace-chroot":         *daemonInplaceChroot,
		"default-config-filename":       flag.DefaultConfigFlagname,
		"disable-cross-origin-requests": *disableCrossOriginRequests,
		"domain":                        config.General.Domain,
		"insecure-ciphers":              config.General.InsecureCiphers,
		"listen-http":                   listenHTTP,
		"listen-https":                  listenHTTPS,
		"listen-proxy":                  listenProxy,
		"listen-https-proxyv2":          listenHTTPSProxyv2,
		"log-format":                    *logFormat,
		"metrics-address":               *metricsAddress,
		"pages-domain":                  *pagesDomain,
		"pages-root":                    *pagesRoot,
		"pages-status":                  *pagesStatus,
		"propagate-correlation-id":      *propagateCorrelationID,
		"redirect-http":                 config.General.RedirectHTTP,
		"root-cert":                     *pagesRootKey,
		"root-key":                      *pagesRootCert,
		"status_path":                   config.General.StatusPath,
		"tls-min-version":               *tlsMinVersion,
		"tls-max-version":               *tlsMaxVersion,
		"use-http-2":                    config.General.HTTP2,
		"gitlab-server":                 config.GitLab.Server,
		"internal-gitlab-server":        config.GitLab.InternalServer,
		"api-secret-key":                *gitLabAPISecretKey,
		"domain-config-source":          config.General.DomainConfigurationSource,
		"use-legacy-storage":            config.General.UseLegacyStorage,
		"auth-redirect-uri":             config.Authentication.RedirectURI,
		"auth-scope":                    config.Authentication.Scope,
		"zip-cache-expiration":          config.Zip.ExpirationInterval,
		"zip-cache-cleanup":             config.Zip.CleanupInterval,
		"zip-cache-refresh":             config.Zip.RefreshInterval,
		"zip-open-timeout":              config.Zip.OpenTimeout,
	}).Debug("Start daemon with configuration")
}

// LoadConfig parses configuration settings passed as command line arguments or
// via config file, and populates a Config object with those values
func LoadConfig() *Config {
	initFlags()

	return loadConfig()
}
