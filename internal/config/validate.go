package config

import (
	"errors"
	"net/url"

	"github.com/hashicorp/go-multierror"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
)

var (
	ErrNoListener                       = errors.New("no listener defined, please specify at least one --listen-* flag")
	ErrAuthNoSecret                     = errors.New("auth-secret must be defined if authentication is supported")
	ErrAuthNoClientID                   = errors.New("auth-client-id must be defined if authentication is supported")
	ErrAuthNoClientSecret               = errors.New("auth-client-secret must be defined if authentication is supported")
	ErrAuthNoGitlabServer               = errors.New("gitlab-server must be defined if authentication is supported")
	ErrAuthNoRedirect                   = errors.New("auth-redirect-uri must be defined if authentication is supported")
	ErrArtifactsServerUnsupportedScheme = errors.New("artifacts-server scheme must be either http:// or https://")
	ErrArtifactsServerInvalidTimeout    = errors.New("artifacts-server-timeout must be greater than or equal to 1")
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
		return ErrNoListener
	}

	return nil
}

func validateAuthConfig(config *Config) error {
	if config.Authentication.Secret == "" && config.Authentication.ClientID == "" &&
		config.Authentication.ClientSecret == "" && config.Authentication.RedirectURI == "" {
		return nil
	}

	var result *multierror.Error
	if config.Authentication.Secret == "" {
		result = multierror.Append(result, ErrAuthNoSecret)
	}
	if config.Authentication.ClientID == "" {
		result = multierror.Append(result, ErrAuthNoClientID)
	}
	if config.Authentication.ClientSecret == "" {
		result = multierror.Append(result, ErrAuthNoClientSecret)
	}
	if config.GitLab.PublicServer == "" {
		result = multierror.Append(result, ErrAuthNoGitlabServer)
	}
	if config.Authentication.RedirectURI == "" {
		result = multierror.Append(result, ErrAuthNoRedirect)
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
		result = multierror.Append(result, ErrArtifactsServerUnsupportedScheme)
	}

	if config.ArtifactsServer.TimeoutSeconds < 1 {
		result = multierror.Append(result, ErrArtifactsServerInvalidTimeout)
	}

	return result.ErrorOrNil()
}
