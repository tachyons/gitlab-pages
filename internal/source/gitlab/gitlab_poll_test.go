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
		interval     time.Duration
		expectedFail bool
	}{
		{
			name:         "success_with_no_retry",
			retries:      0,
			interval:     5 * time.Millisecond,
			expectedFail: false,
		},
		{
			name:         "success_after_N_retries",
			retries:      3,
			interval:     10 * time.Millisecond,
			expectedFail: false,
		},
		{
			name:         "fail_with_no_retries",
			retries:      0,
			interval:     5 * time.Millisecond,
			expectedFail: true,
		},
		{
			name:         "fail_after_N_retries",
			retries:      3,
			interval:     5 * time.Millisecond,
			expectedFail: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer hook.Reset()
			var counter int
			checkerMock := checkerMock{StatusErr: func() error {
				if tt.expectedFail {
					return fmt.Errorf(client.ConnectionErrorMsg)
				}

				if counter < tt.retries {
					counter++
					return fmt.Errorf(client.ConnectionErrorMsg)
				}

				return nil
			}}

			glClient := Gitlab{checker: checkerMock, mu: &sync.RWMutex{}}

			glClient.Poll(tt.retries, tt.interval)
			if tt.expectedFail {
				require.False(t, glClient.isReady)
				s := fmt.Sprintf("polling failed after %d tries every %.2fs", tt.retries+1, tt.interval.Seconds())
				require.Equal(t, s, hook.LastEntry().Message)
				return
			}

			require.True(t, glClient.isReady)
			require.Equal(t, "GitLab internal pages status API connected successfully", hook.LastEntry().Message)

		})
	}
}

type checkerMock struct {
	StatusErr func() error
}

func (c checkerMock) Status() error {
	return c.StatusErr()
}
