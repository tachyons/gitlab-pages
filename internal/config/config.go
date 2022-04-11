package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/namsral/flag"
	"gitlab.com/gitlab-org/labkit/log"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
)

// Config stores all the config options relevant to GitLab Pages.
type Config struct {
	General         General
	RateLimit       RateLimit
	ArtifactsServer ArtifactsServer
	Authentication  Auth
	GitLab          GitLab
	Log             Log
	Sentry          Sentry
	Server          Server
	TLS             TLS
	Zip             ZipServing

	// These fields contain the raw strings passed for listen-http,
	// listen-https, listen-proxy and listen-https-proxyv2 settings. It is used
	// by appmain() to create listeners.
	ListenHTTPStrings         MultiStringFlag
	ListenHTTPSStrings        MultiStringFlag
	ListenProxyStrings        MultiStringFlag
	ListenHTTPSProxyv2Strings MultiStringFlag
}

// General groups settings that are general to GitLab Pages and can not
// be categorized under other head.
type General struct {
	Domain          string
	MaxConns        int
	MaxURILength    int
	MetricsAddress  string
	RedirectHTTP    bool
	RootCertificate []byte
	RootDir         string
	RootKey         []byte
	StatusPath      string

	DisableCrossOriginRequests bool
	InsecureCiphers            bool
	PropagateCorrelationID     bool

	ShowVersion bool

	CustomHeaders []string
}

// RateLimit config struct
type RateLimit struct {
	SourceIPLimitPerSecond float64
	SourceIPBurst          int
	DomainLimitPerSecond   float64
	DomainBurst            int
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

// Cache configuration for GitLab API
type Cache struct {
	CacheExpiry          time.Duration
	CacheCleanupInterval time.Duration
	EntryRefreshTimeout  time.Duration
	RetrievalTimeout     time.Duration
	MaxRetrievalInterval time.Duration
	MaxRetrievalRetries  int
}

// GitLab groups settings related to configuring GitLab client used to
// interact with GitLab API
type GitLab struct {
	PublicServer       string
	InternalServer     string
	APISecretKey       []byte
	ClientHTTPTimeout  time.Duration
	JWTTokenExpiration time.Duration
	Cache              Cache
	EnableDisk         bool
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

type Server struct {
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	ListenKeepAlive   time.Duration
}

func internalGitlabServerFromFlags() string {
	if *internalGitLabServer != "" {
		return *internalGitLabServer
	}

	return *publicGitLabServer
}

func setGitLabAPISecretKey(secretFile string, config *Config) error {
	if secretFile == "" {
		return nil
	}

	encoded, err := os.ReadFile(secretFile)
	if err != nil {
		return fmt.Errorf("reading secret file: %w", err)
	}

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	secretLength, err := base64.StdEncoding.Decode(decoded, encoded)
	if err != nil {
		return fmt.Errorf("decoding GitLab API secret: %w", err)
	}

	if secretLength != 32 {
		return fmt.Errorf("expected 32 bytes GitLab API secret but got %d bytes", secretLength)
	}

	config.GitLab.APISecretKey = decoded
	return nil
}

func loadConfig() (*Config, error) {
	config := &Config{
		General: General{
			Domain:                     strings.ToLower(*pagesDomain),
			MaxConns:                   *maxConns,
			MaxURILength:               *maxURILength,
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
		RateLimit: RateLimit{
			SourceIPLimitPerSecond: *rateLimitSourceIP,
			SourceIPBurst:          *rateLimitSourceIPBurst,
			DomainLimitPerSecond:   *rateLimitDomain,
			DomainBurst:            *rateLimitDomainBurst,
		},
		GitLab: GitLab{
			ClientHTTPTimeout:  *gitlabClientHTTPTimeout,
			JWTTokenExpiration: *gitlabClientJWTExpiry,
			EnableDisk:         *enableDisk,
			Cache: Cache{
				CacheExpiry:          *gitlabCacheExpiry,
				CacheCleanupInterval: *gitlabCacheCleanup,
				EntryRefreshTimeout:  *gitlabCacheRefresh,
				RetrievalTimeout:     *gitlabRetrievalTimeout,
				MaxRetrievalInterval: *gitlabRetrievalInterval,
				MaxRetrievalRetries:  *gitlabRetrievalRetries,
			},
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
		Server: Server{
			ReadTimeout:       *serverReadTimeout,
			ReadHeaderTimeout: *serverReadHeaderTimeout,
			WriteTimeout:      *serverWriteTimeout,
			ListenKeepAlive:   *serverKeepAlive,
		},

		// Actual listener pointers will be populated in appMain. We populate the
		// raw strings here so that they are available in appMain
		ListenHTTPStrings:         listenHTTP,
		ListenHTTPSStrings:        listenHTTPS,
		ListenProxyStrings:        listenProxy,
		ListenHTTPSProxyv2Strings: listenHTTPSProxyv2,
	}

	var err error

	// Populating remaining General settings
	for _, file := range []struct {
		contents *[]byte
		path     string
	}{
		{&config.General.RootCertificate, *pagesRootCert},
		{&config.General.RootKey, *pagesRootKey},
	} {
		if file.path != "" {
			if *file.contents, err = os.ReadFile(file.path); err != nil {
				return nil, err
			}
		}
	}

	// Populating remaining GitLab settings
	config.GitLab.PublicServer = *publicGitLabServer

	config.GitLab.InternalServer = internalGitlabServerFromFlags()

	if err = setGitLabAPISecretKey(*gitLabAPISecretKey, config); err != nil {
		return nil, err
	}

	return config, nil
}

func LogConfig(config *Config) {
	log.WithFields(log.Fields{
		"artifacts-server":              *artifactsServer,
		"artifacts-server-timeout":      *artifactsServerTimeout,
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
		"gitlab-server":                 config.GitLab.PublicServer,
		"internal-gitlab-server":        config.GitLab.InternalServer,
		"api-secret-key":                *gitLabAPISecretKey,
		"enable-disk":                   config.GitLab.EnableDisk,
		"auth-redirect-uri":             config.Authentication.RedirectURI,
		"auth-scope":                    config.Authentication.Scope,
		"max-conns":                     config.General.MaxConns,
		"max-uri-length":                config.General.MaxURILength,
		"zip-cache-expiration":          config.Zip.ExpirationInterval,
		"zip-cache-cleanup":             config.Zip.CleanupInterval,
		"zip-cache-refresh":             config.Zip.RefreshInterval,
		"zip-open-timeout":              config.Zip.OpenTimeout,
		"server-read-timeout":           config.Server.ReadTimeout,
		"server-read-header-timeout":    config.Server.ReadHeaderTimeout,
		"server-write-timeout":          config.Server.WriteTimeout,
		"server-keep-alive":             config.Server.ListenKeepAlive,
	}).Debug("Start Pages with configuration")
}

// LoadConfig parses configuration settings passed as command line arguments or
// via config file, and populates a Config object with those values
func LoadConfig() (*Config, error) {
	initFlags()

	return loadConfig()
}
