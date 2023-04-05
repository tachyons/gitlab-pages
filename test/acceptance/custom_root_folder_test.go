package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestCustomRoot(t *testing.T) {
	RunPagesProcess(t)

	tests := []struct {
		name          string
		requestDomain string
		requestPath   string
		redirectURL   string
		httpStatus    int
	}{
		{
			name:          "custom root",
			requestDomain: "custom-root.gitlab-example.com",
			httpStatus:    http.StatusOK,
		},
		{
			name:          "custom root legacy",
			requestDomain: "custom-root-legacy.gitlab-example.com",
			httpStatus:    http.StatusOK,
		},
		{
			name:          "custom root explicitly public",
			requestDomain: "custom-root-explicit-public.gitlab-example.com",
			httpStatus:    http.StatusOK,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpsListener, test.requestDomain, test.requestPath)
			require.NoError(t, err)
			testhelpers.Close(t, rsp.Body)

			require.Equal(t, test.httpStatus, rsp.StatusCode)
		})
	}
}
