package jail

import (
	"fmt"
	"syscall"

	log "github.com/sirupsen/logrus"
	"golang.org/x/sys/unix"
)

// Unshare makes the process to own it's own mount namespace
// and prevent the changes for mounts to be propagated
// to other mount namespace. Making all mount changes local
// to current process
func (j *Jail) Unshare() error {
	// https://man7.org/linux/man-pages/man7/mount_namespaces.7.html
	err := syscall.Unshare(syscall.CLONE_NEWNS)
	log.WithError(err).Info("unsharing mount namespace")
	if err != nil {
		return err
	}

	// As documented in `mount_namespaces`:
	// An application that creates a new mount namespace directly using
	// clone(2) or unshare(2) may desire to prevent propagation of mount
	// events to other mount namespaces
	err = syscall.Mount("none", "/", "", unix.MS_REC|unix.MS_PRIVATE, "")
	log.WithError(err).Info("changing root filesystem propagation")
	return err
}

func (j *Jail) mount() error {
	for dest, src := range j.bindMounts {
		var opts uintptr = unix.MS_BIND | unix.MS_REC
		if err := unix.Mount(src, dest, "none", opts, ""); err != nil {
			return fmt.Errorf("failed to bind mount %s on %s. %s", src, dest, err)
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
				return fmt.Errorf("failed to unmount %s. %s", dest, err)
			}
		}
	}

	return nil
}
