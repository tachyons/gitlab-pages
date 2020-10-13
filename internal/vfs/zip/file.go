package zip

import (
	"archive/zip"
	"encoding/binary"
	"io"
	"os"
	"sync"

	"gitlab.com/gitlab-org/gitlab-pages/metrics"
)

type zipFile struct {
	*zip.File
}

// recycle removes unneeded metadata from `zip.File` to reduce memory pressure
func (f zipFile) recycle() {
	f.Comment = ""

	// we clean `f.Extra` as we need it for the `cached*` methods
	f.Extra = nil
}

// cachedDataOffset does return a cached offset if present
// or requests and stores it
// it uses `zip.File.Extra` field to hold a data (abusing it)
// as we don't want to allocate more memory
func (f zipFile) cachedDataOffset(lock *sync.RWMutex) (int64, error) {
	if !f.Mode().IsRegular() {
		return 0, errNotFile
	}

	lock.RLock()
	data := f.Extra
	lock.RUnlock()

	if data != nil {
		metrics.ZipCacheRequests.WithLabelValues("data-offset", "hit").Inc()
		return int64(binary.LittleEndian.Uint64(data)), nil
	}

	dataOffset, err := f.DataOffset()
	if err != nil {
		metrics.ZipCacheRequests.WithLabelValues("data-offset", "error").Inc()
		return 0, nil
	}

	encoded := make([]byte, 8)
	binary.LittleEndian.PutUint64(encoded, uint64(dataOffset))
	lock.Lock()
	f.Extra = encoded
	lock.Unlock()

	metrics.ZipCacheRequests.WithLabelValues("data-offset", "miss").Inc()

	return dataOffset, err
}

// cachedReadlink does return a cached offset if present
// or requests and stores it
// it uses `zip.File.Extra` field to hold a data (abusing it)
func (f zipFile) cachedReadlink(lock *sync.RWMutex) (string, error) {
	if f.FileInfo().Mode()&os.ModeSymlink != os.ModeSymlink {
		return "", errNotSymlink
	}

	lock.RLock()
	data := f.Extra
	lock.RUnlock()

	if data != nil {
		metrics.ZipCacheRequests.WithLabelValues("readlink", "hit").Inc()
		return string(f.Extra), nil
	}

	rc, err := f.Open()
	if err != nil {
		metrics.ZipCacheRequests.WithLabelValues("readlink", "error").Inc()
		return "", err
	}
	defer rc.Close()

	// read up to len(symlink) bytes from the link file
	var link [maxSymlinkSize + 1]byte
	n, err := io.ReadFull(rc, link[:])
	if err != nil && err != io.ErrUnexpectedEOF {
		// if err == io.ErrUnexpectedEOF the link is smaller than len(symlink) so it's OK to not return it
		metrics.ZipCacheRequests.WithLabelValues("readlink", "error").Inc()
		return "", err
	}

	lock.Lock()
	f.Extra = link[:n]
	lock.Unlock()

	metrics.ZipCacheRequests.WithLabelValues("readlink", "miss").Inc()
	return string(link[:n]), nil
}
