package customheaders_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/customheaders"
)

func TestAddCustomHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headers     http.Header
		wantHeaders map[string]string
	}{
		{
			name:        "Normal case",
			headers:     http.Header{"X-Test-String": []string{"Test"}},
			wantHeaders: map[string]string{"X-Test-String": "Test"},
		},
		{
			name:        "Non-tracking header case",
			headers:     http.Header{"Tk": []string{"N"}},
			wantHeaders: map[string]string{"Tk": "N"},
		},
		{
			name:        "Content security header case",
			headers:     http.Header{"content-security-policy": []string{"default-src 'self'"}},
			wantHeaders: map[string]string{"Content-Security-Policy": "default-src 'self'"},
		},
		{
			name:        "Multiple header strings",
			headers:     http.Header{"content-security-policy": []string{"default-src 'self'"}, "X-Test-String": []string{"Test"}, "My amazing header": []string{"Amazing"}},
			wantHeaders: map[string]string{"Content-Security-Policy": "default-src 'self'", "X-Test-String": "Test", "My amazing header": "Amazing"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			customheaders.AddCustomHeaders(w, tt.headers)
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
