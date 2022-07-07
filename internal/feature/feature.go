package feature

import "os"

type Feature struct {
	EnvVariable    string
	defaultEnabled bool
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
