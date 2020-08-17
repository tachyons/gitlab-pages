package zipfs

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Attributes struct {
	Name           string
	Size           int64
	CompressedSize int64
	Method         uint16
	os.FileMode
	Idx int
}

func (fs *VFS) readAttributesFromZip() error {
	for i, zf := range fs.zipReader.File {
		name := filepath.Clean(zf.Name)
		const publicPrefix = "public/"
		if !strings.HasPrefix(name, publicPrefix) {
			continue
		}
		name = strings.TrimPrefix(name, publicPrefix)

		mode := zf.FileInfo().Mode()
		switch mode & os.ModeType {
		case 0:
		case os.ModeSymlink:
		default:
			continue
		}

		attr := &Attributes{
			Name:     name,
			Method:   zf.Method,
			FileMode: mode,
			Idx:      i,
		}

		var err error
		attr.Size, err = fitInt64(zf.UncompressedSize64)
		if err != nil {
			return err
		}

		attr.CompressedSize, err = fitInt64(zf.CompressedSize64)
		if err != nil {
			return err
		}

		fs.files = append(fs.files, attr)
	}

	// Sort the attributes so we can use binary search later.
	sort.Slice(fs.files, func(i, j int) bool { return fs.files[i].Name <= fs.files[j].Name })

	return nil
}

func fitInt64(u uint64) (int64, error) {
	if u > math.MaxInt64 {
		return 0, fmt.Errorf("uint64 too large for int64")
	}
	return int64(u), nil
}

func (fs *VFS) getAttributes(name string) (*Attributes, bool) {
	name = filepath.Clean(name)

	i := sort.Search(len(fs.files), func(j int) bool { return fs.files[j].Name >= name })
	if i == len(fs.files) {
		return nil, false
	}

	if attr := fs.files[i]; attr.Name == name {
		return attr, true
	}

	// If name is "foo/bar", see if anything like "foo/bar/X" exists. We
	// could use binary search again but i should be close to a match, if
	// there is one.
	dirPrefix := name + "/"
	for j := i; j < len(fs.files); j++ {
		attr := fs.files[j]
		if n := len(dirPrefix); len(attr.Name) < n || attr.Name[:n] > dirPrefix {
			break
		}

		if strings.HasPrefix(attr.Name, dirPrefix) {
			// "foo/bar/X" exists: return "foo/bar" as a directory.
			return &Attributes{
				Name:     name,
				FileMode: os.ModeDir | 0755,
			}, true
		}
	}

	return nil, false
}
