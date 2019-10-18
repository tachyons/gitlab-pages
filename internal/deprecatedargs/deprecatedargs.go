package deprecatedargs

import (
	"fmt"
	"strings"
)

var deprecatedArgs = []string{"-auth-client-id", "-auth-client-secret", "-auth-secret", "-sentry-dsn"}

// Validate checks if deprecated params have been used
func Validate(args []string) error {
	foundDeprecatedArgs := []string{}
	argMap := make(map[string]bool)

	for _, arg := range args {
		argMap[arg] = true
	}

	for _, deprecatedArg := range deprecatedArgs {
		if argMap[deprecatedArg] {
			foundDeprecatedArgs = append(foundDeprecatedArgs, deprecatedArg)
		}
	}

	if len(foundDeprecatedArgs) > 0 {
		return fmt.Errorf("Deprecation message: %s should not be passed as a command line arguments", strings.Join(foundDeprecatedArgs, ", "))
	}
	return nil
}
