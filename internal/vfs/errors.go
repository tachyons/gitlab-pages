package vfs

import (
	"fmt"
)

type ErrNotExist struct {
	Inner error
}

func (e ErrNotExist) Error() string {
	return fmt.Sprintf("not exist: %q", e.Inner)
}

func IsNotExist(err error) bool {
	_, ok := err.(*ErrNotExist)
	return ok
}
