package testdata

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"

	"gitlab.com/gitlab-org/gitlab-pages/internal/source/gitlab/api"
)

var chdirSet = false

type H struct {
	Host          string
	Path          string
	projectID     int
	accessControl bool
	httpsOnly     bool
}

func FindZipArchiveHandler(t *testing.T) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		domain := r.URL.Query().Get("host")
		if isLocalhost(domain) {
			// shortcut for healthy checkup done by WaitUntilRequestSucceeds
			w.WriteHeader(http.StatusNoContent)
			return
		}

		//    Host:    "CapitalGroup.gitlab-example.com",
		//        path:    "CapitalProject/",
		//        content: "Capital Group & Project\n",
		chdir := testhelpers.ChdirInPath(t, "../../shared/pages", &chdirSet)
		defer chdir()

		wd, err := os.Getwd()
		require.NoError(t, err)
		// $gitlab-pages/shared/pages/Subdomain/Project/public.zip

		zipFilePath := wd + "/" + getSubdomain(t, domain) + "/CapitalProject/public.zip"

		if _, err := os.Stat(zipFilePath); os.IsNotExist(err) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`zipFilePath: ` + zipFilePath))
			return
		}

		vd := api.VirtualDomain{
			LookupPaths: []api.LookupPath{
				{
					ProjectID:     123,
					AccessControl: false,
					HTTPSOnly:     false,
					Prefix:        "/CapitalProject",
					Source: api.Source{
						Type: "zip",
						Path: fmt.Sprintf("file://%s", zipFilePath),
					},
				},
			},
		}

		err = json.NewEncoder(w).Encode(vd)
		require.NoError(t, err)
		return
	}

}

// getSubdomain from a host for example project.gitlab.com returns project
func getSubdomain(t *testing.T, host string) string {
	t.Helper()

	s := strings.Split(host, ".")
	require.GreaterOrEqual(t, len(s), 1)

	return s[0]

}

func isLocalhost(ip string) bool {
	return ip == "127.0.0.1" || ip == "::1"
}
