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
	Server             string
	InternalServer     string
	APISecretKey       []byte
	ClientHTTPTimeout  time.Duration
	JWTTokenExpiration time.Duration
	Cache              Cache
	EnableDisk         bool
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
	// TODO: this is a temporary workaround for https://gitlab.com/gitlab-org/gitlab/-/issues/326117#note_546346101
	// where daemon-inplace-chroot=true fails to serve zip archives when pages_serve_with_zip_file_protocol is enabled
	// To be removed along with chroot support https://gitlab.com/gitlab-org/gitlab-pages/-/issues/561
	ChrootPath string
}

func gitlabServerFromFlags() (string, error) {
	if *gitLabServer != "" {
		return *gitLabServer, nil
	}

	if *gitLabAuthServer != "" {
		log.Warn("auth-server parameter is deprecated, use gitlab-server instead")
		return *gitLabAuthServer, nil
	}

	u, err := url.Parse(*artifactsServer)
	if err != nil {
		return "", fmt.Errorf("parsing artifact server: %w", err)
	}

	u.Path = ""
	return u.String(), nil
}

func internalGitlabServerFromFlags() (string, error) {
	if *internalGitLabServer != "" {
		return *internalGitLabServer, nil
	}

	return gitlabServerFromFlags()
}

func setGitLabAPISecretKey(secretFile string, config *Config) error {
	if secretFile == "" {
		return nil
	}

	encoded, err := ioutil.ReadFile(secretFile)
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
			if *file.contents, err = ioutil.ReadFile(file.path); err != nil {
				return nil, err
			}
		}
	}

	// Populating remaining GitLab settings
	if config.GitLab.Server, err = gitlabServerFromFlags(); err != nil {
		return nil, err
	}

	if config.GitLab.InternalServer, err = internalGitlabServerFromFlags(); err != nil {
		return nil, err
	}

	if err = setGitLabAPISecretKey(*gitLabAPISecretKey, config); err != nil {
		return nil, err
	}

	// TODO: this is a temporary workaround for https://gitlab.com/gitlab-org/gitlab/-/issues/326117#note_546346101
	// where daemon-inplace-chroot=true fails to serve zip archives when pages_serve_with_zip_file_protocol is enabled
	// To be removed along with chroot support https://gitlab.com/gitlab-org/gitlab-pages/-/issues/561
	if config.Daemon.InplaceChroot {
		config.Zip.ChrootPath = *pagesRoot
	}

	if err := validateConfig(config); err != nil {
		return nil, err
	}

	return config, nil
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
		"enable-disk":                   config.GitLab.EnableDisk,
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
func LoadConfig() (*Config, error) {
	initFlags()

	return loadConfig()
}
