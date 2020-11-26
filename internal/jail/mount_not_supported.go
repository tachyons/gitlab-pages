// +build !linux

package jail

import (
	"fmt"
	"runtime"
)

func (j *Jail) Unshare() error {
	return fmt.Errorf("unshare not supported on %s", runtime.GOOS)
}

func (j *Jail) notSupported() error {
	if len(j.bindMounts) > 0 {
		return fmt.Errorf("bind mount not supported on %s", runtime.GOOS)
	}

	return nil
}
func (j *Jail) mount() error {
	return j.notSupported()
}

func (j *Jail) unmount() error {
	return j.notSupported()
}
