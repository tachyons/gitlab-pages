//go:build boringcrypto
// +build boringcrypto

package boring

import (
	"crypto/boring"

	"gitlab.com/gitlab-org/labkit/log"
)

func CheckBoring() {
	if boring.Enabled() {
		log.Info("FIPS mode is enabled. Using BoringSSL.")
		return
	}
	log.Info("GitLab Pages was compiled with FIPS mode but BoringSSL is not enabled.")
}
