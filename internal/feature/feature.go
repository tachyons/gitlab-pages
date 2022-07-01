package feature

import "os"

type Feature struct {
	EnvVariable    string
	defaultEnabled bool
}

// EnforceIPRateLimits enforces IP rate limiter to drop requests
// TODO: remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/706
var EnforceIPRateLimits = Feature{
	EnvVariable: "FF_ENFORCE_IP_RATE_LIMITS",
}

// EnforceDomainRateLimits enforces domain rate limiter to drop requests
// TODO: remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/706
var EnforceDomainRateLimits = Feature{
	EnvVariable: "FF_ENFORCE_DOMAIN_RATE_LIMITS",
}

// EnforceDomainTLSRateLimits enforces domain rate limits on establishing new TLS connections
// TODO: remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/706
var EnforceDomainTLSRateLimits = Feature{
	EnvVariable: "FF_ENFORCE_DOMAIN_TLS_RATE_LIMITS",
}

// EnforceIPTLSRateLimits enforces domain rate limits on establishing new TLS connections
// TODO: remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/706
var EnforceIPTLSRateLimits = Feature{
	EnvVariable: "FF_ENFORCE_IP_TLS_RATE_LIMITS",
}

// RedirectsPlaceholders enables support for placeholders in redirects file
// TODO: remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/620
var RedirectsPlaceholders = Feature{
	EnvVariable:    "FF_ENABLE_PLACEHOLDERS",
	defaultEnabled: true,
}

// HandleReadErrors reports vfs.ReadErrors to sentry and enable error handling
var HandleReadErrors = Feature{
	EnvVariable: "FF_HANDLE_READ_ERRORS",
}

// Enabled reads the environment variable responsible for the feature flag
// if FF is disabled by default, the environment variable needs to be "true" to explicitly enable it
// if FF is enabled by default, variable needs to be "false" to explicitly disable it
func (f Feature) Enabled() bool {
	env := os.Getenv(f.EnvVariable)

	if f.defaultEnabled {
		return env != "false"
	}

	return env == "true"
}
