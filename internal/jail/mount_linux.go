package jail

import (
	"fmt"

	"golang.org/x/sys/unix"
)

func (j *Jail) mount() error {
	for dest, src := range j.bindMounts {
		var opts uintptr = unix.MS_BIND | unix.MS_REC
		if err := unix.Mount(src, dest, "none", opts, ""); err != nil {
			return fmt.Errorf("Failed to bind mount %s on %s. %s", src, dest, err)
		}
	}

	return nil
}

func (j *Jail) unmount() error {
	for dest := range j.bindMounts {
		if err := unix.Unmount(dest, unix.MNT_DETACH); err != nil {
			// A second invocation on unmount with MNT_DETACH flag will return EINVAL
			// there's no need to abort with an error if bind mountpoint is already unmounted
			if err != unix.EINVAL {
				return fmt.Errorf("Failed to unmount %s. %s", dest, err)
			}
		}
	}

	return nil
}
