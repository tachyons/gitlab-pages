package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/hashicorp/go-multierror"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
)

var (
	errNoListener                       = errors.New("no listener defined, please specify at least one --listen-* flag")
	errAuthNoSecret                     = errors.New("auth-secret must be defined if authentication is supported")
	errAuthNoClientID                   = errors.New("auth-client-id must be defined if authentication is supported")
	errAuthNoClientSecret               = errors.New("auth-client-secret must be defined if authentication is supported")
	errAuthNoGitlabServer               = errors.New("gitlab-server must be defined if authentication is supported")
	errAuthNoRedirect                   = errors.New("auth-redirect-uri must be defined if authentication is supported")
	errArtifactsServerUnsupportedScheme = errors.New("artifacts-server scheme must be either http:// or https://")
	errArtifactsServerInvalidTimeout    = errors.New("artifacts-server-timeout must be greater than or equal to 1")
	errEmptyListener                    = errors.New("listener must not be empty")
)

// Validate values populated in Config
func Validate(config *Config) error {
	var result *multierror.Error

	result = multierror.Append(result,
		validateListeners(config),
		validateAuthConfig(config),
		validateArtifactsServerConfig(config),
		tls.ValidateTLSVersions(*tlsMinVersion, *tlsMaxVersion),
	)

	return result.ErrorOrNil()
}

func validateListeners(config *Config) error {
	if config.ListenHTTPStrings.Len() == 0 &&
		config.ListenHTTPSStrings.Len() == 0 &&
		config.ListenHTTPSProxyv2Strings.Len() == 0 &&
		config.ListenProxyStrings.Len() == 0 {
		return errNoListener
	}

	var result *multierror.Error
	for i, s := range config.ListenHTTPStrings.Split() {
		if s == "" {
			result = multierror.Append(result, fmt.Errorf("empty http listener at index %d: %w", i, errEmptyListener))
		}
	}
	for i, s := range config.ListenHTTPSStrings.Split() {
		if s == "" {
			result = multierror.Append(result, fmt.Errorf("empty https listener at index %d: %w", i, errEmptyListener))
		}
	}
	for i, s := range config.ListenHTTPSProxyv2Strings.Split() {
		if s == "" {
			result = multierror.Append(result, fmt.Errorf("empty proxyv2 listener at index %d: %w", i, errEmptyListener))
		}
	}
	for i, s := range config.ListenProxyStrings.Split() {
		if s == "" {
			result = multierror.Append(result, fmt.Errorf("empty proxy listener at index %d: %w", i, errEmptyListener))
		}
	}

	return result.ErrorOrNil()
}

func validateAuthConfig(config *Config) error {
	if config.Authentication.Secret == "" && config.Authentication.ClientID == "" &&
		config.Authentication.ClientSecret == "" && config.Authentication.RedirectURI == "" {
		return nil
	}

	var result *multierror.Error
	if config.Authentication.Secret == "" {
		result = multierror.Append(result, errAuthNoSecret)
	}
	if config.Authentication.ClientID == "" {
		result = multierror.Append(result, errAuthNoClientID)
	}
	if config.Authentication.ClientSecret == "" {
		result = multierror.Append(result, errAuthNoClientSecret)
	}
	if config.GitLab.PublicServer == "" {
		result = multierror.Append(result, errAuthNoGitlabServer)
	}
	if config.Authentication.RedirectURI == "" {
		result = multierror.Append(result, errAuthNoRedirect)
	}
	return result.ErrorOrNil()
}

func validateArtifactsServerConfig(config *Config) error {
	if config.ArtifactsServer.URL == "" {
		return nil
	}

	u, err := url.Parse(config.ArtifactsServer.URL)
	if err != nil {
		return err
	}

	var result *multierror.Error

	// url.Parse ensures that the Scheme attribute is always lower case.
	if u.Scheme != "http" && u.Scheme != "https" {
		result = multierror.Append(result, errArtifactsServerUnsupportedScheme)
	}

	if config.ArtifactsServer.TimeoutSeconds < 1 {
		result = multierror.Append(result, errArtifactsServerInvalidTimeout)
	}

	return result.ErrorOrNil()
}
