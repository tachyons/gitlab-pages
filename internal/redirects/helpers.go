package redirects

import (
	"os"
	"strings"
)

// normalizePath ensures that the provided path ends with at least one trailing slash.
// Multiple trailing slashs are not trimmed.
func normalizePath(path string) string {
	return strings.TrimSuffix(path, "/") + "/"
}

// placeholdersEnabled returns whether or not placeholders and splats
// are enabled for use in the _redirects file.
func placeholdersEnabled() bool {
	return os.Getenv(FFEnablePlaceholders) == "true"
}
