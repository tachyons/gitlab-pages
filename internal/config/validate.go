package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/hashicorp/go-multierror"
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
		validateTLSVersions(*tlsMinVersion, *tlsMaxVersion),
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

	result = multierror.Append(result,
		validateListenerAddr(config.ListenHTTPStrings, "http"),
		validateListenerAddr(config.ListenHTTPSStrings, "https"),
		validateListenerAddr(config.ListenHTTPSProxyv2Strings, "proxyv2"),
		validateListenerAddr(config.ListenProxyStrings, "proxy"),
	)

	return result.ErrorOrNil()
}

func validateListenerAddr(listeners MultiStringFlag, name string) error {
	var result *multierror.Error
	for i, s := range listeners.Split() {
		if s == "" {
			result = multierror.Append(result, fmt.Errorf("empty %s listener at index %d: %w", name, i, errEmptyListener))
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

// validateTLSVersions returns error if the provided TLS versions config values are not valid
func validateTLSVersions(min, max string) error {
	tlsMin, tlsMinOk := allTLSVersions[min]
	tlsMax, tlsMaxOk := allTLSVersions[max]

	if !tlsMinOk {
		return fmt.Errorf("invalid minimum TLS version: %s", min)
	}
	if !tlsMaxOk {
		return fmt.Errorf("invalid maximum TLS version: %s", max)
	}
	if tlsMin > tlsMax && tlsMax > 0 {
		return fmt.Errorf("invalid maximum TLS version: %s; should be at least %s", max, min)
	}

	return nil
}
