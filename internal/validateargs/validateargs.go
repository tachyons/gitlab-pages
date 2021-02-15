package validateargs

import (
	"fmt"
	"strings"
)

const (
	deprecatedMessage = "command line options have been deprecated:"
	notAllowedMsg     = "invalid command line arguments:"
)

var deprecatedArgs = []string{"-sentry-dsn"}
var notAllowedArgs = []string{"-auth-client-id", "-auth-client-secret", "-auth-secret", "-auth-scope"}

// Deprecated checks if deprecated params have been used
func Deprecated(args []string) error {
	return validate(args, deprecatedArgs, deprecatedMessage)
}

// NotAllowed checks if explicitly not allowed params have been used
func NotAllowed(args []string) error {
	return validate(args, notAllowedArgs, notAllowedMsg)
}

func validate(args, invalidArgs []string, errMsg string) error {
	var foundInvalidArgs []string

	argsStr := strings.Join(args, " ")
	for _, invalidArg := range invalidArgs {
		if strings.Contains(argsStr, invalidArg) {
			foundInvalidArgs = append(foundInvalidArgs, invalidArg)
		}
	}

	if len(foundInvalidArgs) > 0 {
		return fmt.Errorf("%s %s", errMsg, strings.Join(foundInvalidArgs, ", "))
	}

	return nil
}
