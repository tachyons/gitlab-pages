// +build go1.13,!go1.14

package tlsconfig

import (
	"fmt"
	"os"
	"strings"
)

func init() {
	validateGoDebug = func() error {
		if strings.Contains(os.Getenv("GODEBUG"), "tls13=0") {
			return fmt.Errorf("tls1.3 is disabled: GODEBUG=%s", os.Getenv("GODEBUG"))
		}

		return nil
	}
}
