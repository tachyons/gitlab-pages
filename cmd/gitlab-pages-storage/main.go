package main

import (
	"archive/zip"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	_ "gocloud.dev/blob/fileblob"

	"gitlab.com/gitlab-org/gitlab-pages/internal/vfs/gocloud"
)

const (
	usage = "Usage: gitlab-pages-storage upload ZIP_FILE ZIP_PREFIX BUCKET_URL BUCKET_PREFIX"
)

func main() {
	args := os.Args[1:]
	if len(args) != 5 || args[0] != "upload" {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}

	if err := upload(args[1], args[2], args[3], args[4]); err != nil {
		fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func upload(zipPath string, zipPrefix string, bucketURL string, bucketPrefix string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	zr, zipClose, err := openZip(zipPath)
	if err != nil {
		return err
	}
	defer zipClose()

	zipPrefix, err = cleanPrefix(zipPrefix)
	if err != nil {
		return err
	}

	bucketPrefix, err = cleanPrefix(bucketPrefix)
	if err != nil {
		return err
	}

	fs, err := gocloud.New(ctx, bucketURL, bucketPrefix)
	if err != nil {
		return err
	}

	return uploadEntries(ctx, zr, fs, zipPrefix)
}

func uploadEntries(ctx context.Context, zr *zip.Reader, fs *gocloud.VFS, zipPrefix string) error {
	for _, zf := range zr.File {
		key := filepath.Clean(zf.Name)
		if !strings.HasPrefix(key, zipPrefix) {
			continue
		}

		switch zf.Mode() & os.ModeType {
		case 0: // regular file
			if err := handleFile(ctx, fs, key, zf); err != nil {
				return err
			}
		case os.ModeSymlink:
			if err := handleSymlink(ctx, fs, key, zf, zipPrefix); err != nil {
				return err
			}
		default:
			// skip anything that is not a file or a symlink
		}
	}

	return nil
}

func openZip(zipPath string) (*zip.Reader, func() error, error) {
	f, err := os.Open(zipPath)
	if err != nil {
		return nil, nil, err
	}

	fi, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, nil, err
	}

	zf, err := zip.NewReader(f, fi.Size())
	if err != nil {
		f.Close()
		return nil, nil, err
	}

	return zf, f.Close, err
}

func handleFile(ctx context.Context, fs *gocloud.VFS, key string, zf *zip.File) error {
	r, err := zf.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	return fs.WriteFile(ctx, key, r)
}

func handleSymlink(ctx context.Context, fs *gocloud.VFS, key string, zf *zip.File, prefix string) error {
	r, err := zf.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	targetBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	target := string(targetBytes)
	if !strings.HasPrefix(filepath.Join(filepath.Dir(key), target), prefix) {
		fmt.Fprintf(os.Stderr, "warning: ignoring symlink %q pointing outside root prefix\n", key)
		return nil
	}

	return fs.WriteSymlink(ctx, key, target)
}

func cleanPrefix(prefix string) (string, error) {
	prefix = filepath.Clean(prefix)
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	if strings.HasPrefix(prefix, "../") {
		return "", fmt.Errorf("invalid prefix: %s", prefix)
	}

	return prefix, nil
}
