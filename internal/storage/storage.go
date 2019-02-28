package storage

import (
	"io"
	"os"

	"gitlab.com/gitlab-org/gitlab-pages/internal/client"
)

type File interface {
	io.Reader
	io.Seeker
	io.Closer
}

type S interface {
	Resolve(path string) (string, error)
	Stat(path string) (os.FileInfo, error)
	Open(path string) (File, os.FileInfo, error)
}

func New(lookupPath *client.LookupPath) S {
	return &fileSystem{lookupPath}
}
