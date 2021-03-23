package config

import (
	"net/url"

	log "github.com/sirupsen/logrus"

	"gitlab.com/gitlab-org/gitlab-pages/internal/config/tls"
)

func validateConfig(config *Config) {
	if config.General.RootDir == "false" && config.General.DomainConfigurationSource == "disk" {
		log.Fatal("incompatible settings for pages-root=false and domain-config-source=disk, either use domain-config-source=gitlab or set a valid pages-root")
	}

	validateAuthConfig(config)
	validateArtifactsServerConfig(config)
	validateTLSConfig()
}

func validateAuthConfig(config *Config) {
	if config.Authentication.Secret == "" && config.Authentication.ClientID == "" &&
		config.Authentication.ClientSecret == "" && config.Authentication.RedirectURI == "" {
		return
	}

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

func validateArtifactsServerConfig(config *Config) {
	if config.ArtifactsServer.URL == "" {
		return
	}

	u, err := url.Parse(config.ArtifactsServer.URL)
	if err != nil {
		log.Fatal(err)
	}
	// url.Parse ensures that the Scheme attribute is always lower case.
	if u.Scheme != "http" && u.Scheme != "https" {
		log.Fatal("artifacts-server scheme must be either http:// or https://")
	}

	if config.ArtifactsServer.TimeoutSeconds < 1 {
		log.Fatal("artifacts-server-timeout must be greater than or equal to 1")
	}
}

func validateTLSConfig() {
	if err := tls.ValidateTLSVersions(*tlsMinVersion, *tlsMaxVersion); err != nil {
		fatal(err, "invalid TLS version")
	}
}
