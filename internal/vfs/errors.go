package vfs

import "fmt"

const message = "An error occurred while trying to read vfs.File content"

type ReadError struct {
	wrapped error
}

func NewReadError(wrapped error) *ReadError {
	return &ReadError{
		wrapped: wrapped,
	}
}

func (r *ReadError) Error() string {
	if r.wrapped == nil {
		return message
	}

	return fmt.Sprintf("%s: %s", message, r.wrapped.Error())
}

func (r *ReadError) Unwrap() error {
	return r.wrapped
}

func (r *ReadError) Is(target error) bool {
	// nolint: errorlint // implementing type equality for errors.Is
	_, ok := target.(*ReadError)
	return ok
}
