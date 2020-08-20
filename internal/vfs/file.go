package vfs

import "io"

// File represents an open file, which will typically be the response body of a Pages request.
type File interface {
	io.Reader
	io.Seeker
	io.Closer
}
