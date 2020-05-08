package validateargs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidParams(t *testing.T) {
	args := []string{"gitlab-pages",
		"-listen-http", ":3010",
		"-artifacts-server", "http://192.168.1.123:3000/api/v4",
		"-pages-domain", "127.0.0.1.xip.io"}
	require.NoError(t, Deprecated(args))
	require.NoError(t, NotAllowed(args))
}

func TestInvalidDeprecatedParms(t *testing.T) {
	tests := map[string][]string{
		"Sentry DSN passed":          []string{"gitlab-pages", "-sentry-dsn", "abc123"},
		"Sentry DSN using key=value": []string{"gitlab-pages", "-sentry-dsn=abc123"},
	}

	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			err := Deprecated(args)
			require.Error(t, err)
			require.Contains(t, err.Error(), deprecatedMessage)
		})
	}
}

func TestInvalidNotAllowedParams(t *testing.T) {
	tests := map[string][]string{
		"Client ID passed":     []string{"gitlab-pages", "-auth-client-id", "abc123"},
		"Client secret passed": []string{"gitlab-pages", "-auth-client-secret", "abc123"},
		"Auth secret passed":   []string{"gitlab-pages", "-auth-secret", "abc123"},
		"Multiple keys passed": []string{"gitlab-pages", "-auth-client-id", "abc123", "-auth-client-secret", "abc123"},
		"key=value":            []string{"gitlab-pages", "-auth-client-id=abc123"},
		"multiple key=value":   []string{"gitlab-pages", "-auth-client-id=abc123", "-auth-client-secret=abc123"},
	}

	for name, args := range tests {
		t.Run(name, func(t *testing.T) {
			err := NotAllowed(args)
			require.Error(t, err)
			require.Contains(t, err.Error(), notAllowedMsg)
		})
	}
}
