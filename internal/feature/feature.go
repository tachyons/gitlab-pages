package feature

import "os"

type Feature struct {
	EnvVariable    string
	defaultEnabled bool
}

// EnforceIPRateLimits enforces IP rate limiter to drop requests
// TODO: remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/629
var EnforceIPRateLimits = Feature{
	EnvVariable: "FF_ENFORCE_IP_RATE_LIMITS",
}

// EnforceDomainRateLimits enforces domain rate limiter to drop requests
// TODO: remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/655
var EnforceDomainRateLimits = Feature{
	EnvVariable: "FF_ENFORCE_DOMAIN_RATE_LIMITS",
}

// RedirectsPlaceholders enables support for placeholders in redirects file
// TODO: remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/620
var RedirectsPlaceholders = Feature{
	EnvVariable: "FF_ENABLE_PLACEHOLDERS",
}

// HandleCacheHeaders enables handling cache headers when serving from compressed ZIP archives
// TODO: enable and remove https://gitlab.com/gitlab-org/gitlab-pages/-/issues/672
var HandleCacheHeaders = Feature{
	EnvVariable: "FF_HANDLE_CACHE_HEADERS",
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
