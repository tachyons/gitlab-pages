package gitlab

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/client"
)

func TestClient_Poll(t *testing.T) {
	hook := test.NewGlobal()
	tests := []struct {
		name         string
		retries      int
		maxTime      time.Duration
		expectedFail bool
	}{
		{
			name:         "success_with_no_retry",
			retries:      0,
			maxTime:      10 * time.Millisecond,
			expectedFail: false,
		},
		{
			name:         "success_after_N_retries",
			retries:      3,
			maxTime:      30 * time.Millisecond,
			expectedFail: false,
		},
		{
			name:         "fail_with_no_retries",
			retries:      0,
			maxTime:      10 * time.Millisecond,
			expectedFail: true,
		},
		{
			name:         "fail_after_N_retries",
			retries:      3,
			maxTime:      30 * time.Millisecond,
			expectedFail: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer hook.Reset()
			var counter int
			client := client.StubClient{StatusErr: func() error {
				if tt.expectedFail {
					return fmt.Errorf(client.ConnectionErrorMsg)
				}

				if counter < tt.retries {
					counter++
					return fmt.Errorf(client.ConnectionErrorMsg)
				}

				return nil
			}}

			glClient := Gitlab{client: client, mu: &sync.RWMutex{}}

			glClient.poll(3*time.Millisecond, tt.maxTime)
			if tt.expectedFail {
				require.False(t, glClient.isReady)

				s := fmt.Sprintf("Failed to connect to the internal GitLab API after %.2fs", tt.maxTime.Seconds())
				require.Equal(t, s, hook.LastEntry().Message)
				return
			}

			require.True(t, glClient.isReady)
			require.Equal(t, "GitLab internal pages status API connected successfully", hook.LastEntry().Message)
		})
	}
}
