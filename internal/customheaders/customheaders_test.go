package customheaders_test

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/customheaders"
)

func TestParseHeaderString(t *testing.T) {
	tests := []struct {
		name          string
		headerStrings []string
		valid         bool
		expectedLen   int
	}{
		{
			name:          "Normal case",
			headerStrings: []string{"X-Test-String: Test"},
			valid:         true,
			expectedLen:   1,
		},
		{
			name:          "Non-tracking header case",
			headerStrings: []string{"Tk: N"},
			valid:         true,
			expectedLen:   1,
		},
		{
			name:          "Content security header case",
			headerStrings: []string{"content-security-policy: default-src 'self'"},
			valid:         true,
			expectedLen:   1,
		},
		{
			name:          "Multiple header strings",
			headerStrings: []string{"content-security-policy: default-src 'self'", "X-Test-String: Test", "My amazing header : Amazing"},
			valid:         true,
			expectedLen:   3,
		},
		{
			name:          "Multiple invalid cases",
			headerStrings: []string{"content-security-policy: default-src 'self'", "test-case"},
			valid:         false,
		},
		{
			name:          "Not valid case",
			headerStrings: []string{"Tk= N"},
			valid:         false,
		},
		{
			name:          "duplicate headers",
			headerStrings: []string{"Tk: N", "Tk: M"},
			valid:         false,
		},
		{
			name:          "Not valid case",
			headerStrings: []string{"X-Test-String Some-Test"},
			valid:         false,
		},
		{
			name:          "Valid and not valid case",
			headerStrings: []string{"content-security-policy: default-src 'self'", "test-case"},
			valid:         false,
		},
		{
			name:          "Multiple headers in single string parsed as one header",
			headerStrings: []string{"content-security-policy: default-src 'self',X-Test-String: Test,My amazing header : Amazing"},
			valid:         true,
			expectedLen:   1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := customheaders.ParseHeaderString(tt.headerStrings)
			if tt.valid {
				require.NoError(t, err)
				require.Len(t, got, tt.expectedLen)
				return
			}

			require.Error(t, err)
		})
	}
}

func TestAddCustomHeaders(t *testing.T) {
	tests := []struct {
		name          string
		headerStrings []string
		wantHeaders   map[string]string
	}{
		{
			name:          "Normal case",
			headerStrings: []string{"X-Test-String: Test"},
			wantHeaders:   map[string]string{"X-Test-String": "Test"},
		},
		{
			name:          "Non-tracking header case",
			headerStrings: []string{"Tk: N"},
			wantHeaders:   map[string]string{"Tk": "N"},
		},
		{
			name:          "Content security header case",
			headerStrings: []string{"content-security-policy: default-src 'self'"},
			wantHeaders:   map[string]string{"Content-Security-Policy": "default-src 'self'"},
		},
		{
			name:          "Multiple header strings",
			headerStrings: []string{"content-security-policy: default-src 'self'", "X-Test-String: Test", "My amazing header: Amazing"},
			wantHeaders:   map[string]string{"Content-Security-Policy": "default-src 'self'", "X-Test-String": "Test", "My amazing header": "Amazing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers, err := customheaders.ParseHeaderString(tt.headerStrings)
			require.NoError(t, err)
			w := httptest.NewRecorder()
			customheaders.AddCustomHeaders(w, headers)
			rsp := w.Result()
			for k, v := range tt.wantHeaders {
				require.Len(t, rsp.Header[k], 1)

				// use the map directly to make sure ParseHeaderString is adding the canonical keys
				got := rsp.Header[k][0]
				require.Equal(t, v, got, "Expected header %+v, got %+v", v, got)
			}
		})
	}
}
