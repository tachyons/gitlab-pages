package validateargs

import (
	"fmt"
	"strings"
)

var deprecatedArgs = []string{"-sentry-dsn"}
var notAllowedArgs = []string{"-auth-client-id", "-auth-client-secret", "-auth-secret"}

// Deprecated checks if deprecated params have been used
func Deprecated(args []string) error {
	var foundDeprecatedArgs []string

	argsStr := strings.Join(args, " ")
	for _, deprecatedArg := range deprecatedArgs {
		if strings.Contains(argsStr, deprecatedArg) {
			foundDeprecatedArgs = append(foundDeprecatedArgs, deprecatedArg)
		}
	}

	if len(foundDeprecatedArgs) > 0 {
		return fmt.Errorf("deprecation message: %s should not be passed as a command line arguments", strings.Join(foundDeprecatedArgs, ", "))
	}
	return nil
}

// NotAllowed checks if explicitly not allowed params have been used
func NotAllowed(args []string) error {
	var foundNotAllowedArgs []string

	argsStr := strings.Join(args, " ")
	for _, notAllowedArg := range notAllowedArgs {
		if strings.Contains(argsStr, notAllowedArg) {
			foundNotAllowedArgs = append(foundNotAllowedArgs, notAllowedArg)
		}
	}

	if len(foundNotAllowedArgs) > 0 {
		return fmt.Errorf("%s should not be passed as a command line arguments", strings.Join(foundNotAllowedArgs, ", "))
	}

	return nil
}
