package acceptance_test

import (
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAcmeChallengesWhenItIsNotConfigured(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "")
	defer teardown()

	t.Run("When domain folder contains requested acme challenge it responds with it", func(t *testing.T) {
		rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
			existingAcmeTokenPath)

		defer rsp.Body.Close()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rsp.StatusCode)
		body, _ := ioutil.ReadAll(rsp.Body)
		require.Equal(t, "this is token\n", string(body))
	})

	t.Run("When domain folder doesn't contains requested acme challenge it returns 404",
		func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
				notExistingAcmeTokenPath)

			defer rsp.Body.Close()
			require.NoError(t, err)
			require.Equal(t, http.StatusNotFound, rsp.StatusCode)
		},
	)
}

func TestAcmeChallengesWhenItIsConfigured(t *testing.T) {
	skipUnlessEnabled(t)

	teardown := RunPagesProcess(t, *pagesBinary, listeners, "", "-gitlab-server=https://gitlab-acme.com")
	defer teardown()

	t.Run("When domain folder contains requested acme challenge it responds with it", func(t *testing.T) {
		rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
			existingAcmeTokenPath)

		defer rsp.Body.Close()
		require.NoError(t, err)
		require.Equal(t, http.StatusOK, rsp.StatusCode)
		body, _ := ioutil.ReadAll(rsp.Body)
		require.Equal(t, "this is token\n", string(body))
	})

	t.Run("When domain folder doesn't contains requested acme challenge it redirects to GitLab",
		func(t *testing.T) {
			rsp, err := GetRedirectPage(t, httpListener, "withacmechallenge.domain.com",
				notExistingAcmeTokenPath)

			defer rsp.Body.Close()
			require.NoError(t, err)
			require.Equal(t, http.StatusTemporaryRedirect, rsp.StatusCode)

			url, err := url.Parse(rsp.Header.Get("Location"))
			require.NoError(t, err)

			require.Equal(t, url.String(), "https://gitlab-acme.com/-/acme-challenge?domain=withacmechallenge.domain.com&token=notexistingtoken")
		},
	)
}
