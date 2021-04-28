package config

import (
	"errors"
	"net/url"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
)

func validateConfig(config *Config) error {
	if err := validateAuthConfig(config); err != nil {
		return err
	}

	if err := validateArtifactsServerConfig(config); err != nil {
		return err
	}

	return tls.ValidateTLSVersions(*tlsMinVersion, *tlsMaxVersion)
}

func validateAuthConfig(config *Config) error {
	if config.Authentication.Secret == "" && config.Authentication.ClientID == "" &&
		config.Authentication.ClientSecret == "" && config.Authentication.RedirectURI == "" {
		return nil
	}

	if config.Authentication.Secret == "" {
		return errors.New("auth-secret must be defined if authentication is supported")
	}
	if config.Authentication.ClientID == "" {
		return errors.New("auth-client-id must be defined if authentication is supported")
	}
	if config.Authentication.ClientSecret == "" {
		return errors.New("auth-client-secret must be defined if authentication is supported")
	}
	if config.GitLab.Server == "" {
		return errors.New("gitlab-server must be defined if authentication is supported")
	}
	if config.Authentication.RedirectURI == "" {
		return errors.New("auth-redirect-uri must be defined if authentication is supported")
	}
	return nil
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
