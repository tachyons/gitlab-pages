package middleware

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParseHeaderString(t *testing.T) {
	tests := []struct {
		name          string
		headerStrings []string
		valid         bool
	}{{
		name:          "Normal case",
		headerStrings: []string{"X-Test-String: Test"},
		valid:         true,
	},
		{
			name:          "Whitespace trim case",
			headerStrings: []string{"   X-Test-String   :   Test  "},
			valid:         true,
		},
		{
			name:          "Whitespace in key, value case",
			headerStrings: []string{"My amazing header: This is a test"},
			valid:         true,
		},
		{
			name:          "Non-tracking header case",
			headerStrings: []string{"Tk: N"},
			valid:         true,
		},
		{
			name:          "Content security header case",
			headerStrings: []string{"content-security-policy: default-src 'self'"},
			valid:         true,
		},
		{
			name:          "Multiple header strings",
			headerStrings: []string{"content-security-policy: default-src 'self'", "X-Test-String: Test", "My amazing header : Amazing"},
			valid:         true,
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
			name:          "Multiple headers in single string",
			headerStrings: []string{"content-security-policy: default-src 'self',X-Test-String: Test,My amazing header : Amazing"},
			valid:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseHeaderString(tt.headerStrings)
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestAddCustomHeaders(t *testing.T) {
	tests := []struct {
		name          string
		headerStrings []string
		wantHeaders   map[string]string
	}{{
		name:          "Normal case",
		headerStrings: []string{"X-Test-String: Test"},
		wantHeaders:   map[string]string{"X-Test-String": "Test"},
	},
		{
			name:          "Whitespace trim case",
			headerStrings: []string{"   X-Test-String   :   Test  "},
			wantHeaders:   map[string]string{"X-Test-String": "Test"},
		},
		{
			name:          "Whitespace in key, value case",
			headerStrings: []string{"My amazing header: This is a test"},
			wantHeaders:   map[string]string{"My amazing header": "This is a test"},
		},
		{
			name:          "Non-tracking header case",
			headerStrings: []string{"Tk: N"},
			wantHeaders:   map[string]string{"Tk": "N"},
		},
		{
			name:          "Content security header case",
			headerStrings: []string{"content-security-policy: default-src 'self'"},
			wantHeaders:   map[string]string{"content-security-policy": "default-src 'self'"},
		},
		{
			name:          "Multiple header strings",
			headerStrings: []string{"content-security-policy: default-src 'self'", "X-Test-String: Test", "My amazing header : Amazing"},
			wantHeaders:   map[string]string{"content-security-policy": "default-src 'self'", "X-Test-String": "Test", "My amazing header": "Amazing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			headers, err := ParseHeaderString(tt.headerStrings)
			require.NoError(t, err)
			w := httptest.NewRecorder()
			AddCustomHeaders(w, headers)
			for k, v := range tt.wantHeaders {
				require.Equal(t, v, w.HeaderMap.Get(k), "Expected header %+v, got %+v", v, w.HeaderMap.Get(k))
			}
		})
	}
}
