package storage

import (
	"archive/zip"
	"errors"
	"os"

	"gitlab.com/gitlab-org/gitlab-pages/internal/client"
)

type zipStorage struct {
	*client.LookupPath

	archive *zip.ReadCloser
}

func (z *zipStorage) Resolve(path string) (string, error) {
	return "", errors.New("not supported")
}

func (z *zipStorage) Stat(path string) (os.FileInfo, error) {
	return nil, errors.New("not supported")
}

func (z *zipStorage) Open(path string) (File, os.FileInfo, error) {
	return nil, nil, errors.New("not supported")
}

func (z *zipStorage) Close() {
	z.archive.Close()
}

func newZipStorage(lookupPath *client.LookupPath) (S, error) {
	archive, err := zip.OpenReader(lookupPath.ArchivePath)
	if err != nil {
		return nil, err
	}

	return &zipStorage{LookupPath: lookupPath, archive: archive}, nil
}
