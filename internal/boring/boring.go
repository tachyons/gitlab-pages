//go:build boringcrypto
// +build boringcrypto

package boring

import "gitlab.com/gitlab-org/labkit/log"

func CheckBoring() {
	log.Info("FIPS mode is enabled. Using BoringSSL.")
}
