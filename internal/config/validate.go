package config

import (
	"errors"
	"net/url"

	"github.com/hashicorp/go-multierror"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
)

// Validate values populated in Config
func Validate(config *Config) error {
	if err := validateListeners(config); err != nil {
		return err
	}

	if err := validateAuthConfig(config); err != nil {
		return err
	}

	if err := validateArtifactsServerConfig(config); err != nil {
		return err
	}

	return tls.ValidateTLSVersions(*tlsMinVersion, *tlsMaxVersion)
}

func validateListeners(config *Config) error {
	if config.ListenHTTPStrings.Len() == 0 &&
		config.ListenHTTPSStrings.Len() == 0 &&
		config.ListenHTTPSProxyv2Strings.Len() == 0 &&
		config.ListenProxyStrings.Len() == 0 {
		return errors.New("no listener defined, please specify at least one --listen-* flag")
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
		err := errors.New("auth-secret must be defined if authentication is supported")
		result = multierror.Append(result, err)
	}
	if config.Authentication.ClientID == "" {
		err := errors.New("auth-client-id must be defined if authentication is supported")
		result = multierror.Append(result, err)
	}
	if config.Authentication.ClientSecret == "" {
		err := errors.New("auth-client-secret must be defined if authentication is supported")
		result = multierror.Append(result, err)
	}
	if config.GitLab.PublicServer == "" {
		err := errors.New("gitlab-server must be defined if authentication is supported")
		result = multierror.Append(result, err)
	}
	if config.Authentication.RedirectURI == "" {
		err := errors.New("auth-redirect-uri must be defined if authentication is supported")
		result = multierror.Append(result, err)
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
	// url.Parse ensures that the Scheme attribute is always lower case.
	if u.Scheme != "http" && u.Scheme != "https" {
		return errors.New("artifacts-server scheme must be either http:// or https://")
	}

	if config.ArtifactsServer.TimeoutSeconds < 1 {
		return errors.New("artifacts-server-timeout must be greater than or equal to 1")
	}

	return nil
}
