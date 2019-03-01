package storage

import (
	"errors"
	"io"
	"os"

	"gitlab.com/gitlab-org/gitlab-pages/internal/client"
)

// File provides a basic required interface
// to interact with the file, to read, stat, and seek
type File interface {
	io.Reader
	//io.Seeker
	io.Closer
}

// S provides a basic interface to resolve and read files
// from the storage
type S interface {
	Resolve(path string) (string, error)
	Stat(path string) (os.FileInfo, error)
	Open(path string) (File, os.FileInfo, error)
	Close()
}

// New provides a compatible storage with lookupPath
func New(lookupPath *client.LookupPath) (S, error) {
	if lookupPath.Path != "" {
		return &fileSystem{lookupPath}, nil
	} else if lookupPath.ArchivePath != "" {
		return newZipStorage(lookupPath)
	} else {
		return nil, errors.New("storage not supported")
	}
}
