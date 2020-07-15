package client

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClient_Poll(t *testing.T) {
	tests := []struct {
		name     string
		retries  int
		interval time.Duration
		timeout  time.Duration
		status   int
		wantErr  bool
	}{
		{
			name:     "success_with_no_retry",
			retries:  0,
			interval: 5 * time.Millisecond,
			timeout:  5 * time.Millisecond,
			status:   http.StatusOK,
			wantErr:  false,
		},
		{
			name:     "success_after_N_retries",
			retries:  3,
			interval: 5 * time.Millisecond,
			timeout:  20 * time.Millisecond,
			status:   http.StatusOK,
			wantErr:  false,
		},
		{
			name:     "fail_with_no_retries",
			retries:  0,
			interval: 5 * time.Millisecond,
			timeout:  100 * time.Millisecond,
			status:   http.StatusUnauthorized,
			wantErr:  true,
		},
		{
			name:     "fail_after_N_retries",
			retries:  3,
			interval: 5 * time.Millisecond,
			timeout:  100 * time.Millisecond,
			status:   http.StatusUnauthorized,
			wantErr:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var counter int
			mux := http.NewServeMux()
			mux.HandleFunc("/api/v4/internal/pages/status", func(w http.ResponseWriter, r *http.Request) {
				if counter < tt.retries {
					counter++
					// fail on purpose until we reach the max retry
					w.WriteHeader(http.StatusInternalServerError)
					return
				}

				w.WriteHeader(tt.status)
			})

			server := httptest.NewServer(mux)
			defer server.Close()

			client := defaultClient(t, server.URL)
			errCh := make(chan error)

			go client.Poll(tt.retries, tt.interval, errCh)

			// go func() {
			select {
			case err := <-errCh:
				if tt.wantErr {
					require.Error(t, err)
					require.Contains(t, err.Error(), "polling failed after")
					require.Contains(t, err.Error(), ConnectionErrorMsg)
					return
				}
				require.NoError(t, err)
			case <-time.After(tt.timeout):
				t.Logf("%s timed out", tt.name)
				t.FailNow()
			}
			// }()
		})
	}
}
