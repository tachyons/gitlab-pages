package redirects

import (
	"strings"
)

// normalizePath ensures that the provided path ends with at least one trailing slash.
// Multiple trailing slashs are not trimmed.
func normalizePath(path string) string {
	return strings.TrimSuffix(path, "/") + "/"
}
