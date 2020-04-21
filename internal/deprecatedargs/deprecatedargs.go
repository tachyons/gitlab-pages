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
		keyValue := strings.Split(arg, "=")
		if len(keyValue) >= 1 {
			argMap[keyValue[0]] = true
		} else {
			argMap[arg] = true
		}
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
