package deploy

import (
	"archive/zip"
	"errors"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const extractPrefix = "public/"

var ErrNoPublicFiles = errors.New("error: archive has no files in public/")

func ExtractZip(archivePath string, extractPath string) error {
	archive, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer archive.Close()

	extracted := 0
	for _, f := range archive.File {
		fi := f.FileInfo()
		if fi.IsDir() {
			continue
		}

		cleanName := filepath.Clean(f.Name)
		if !strings.HasPrefix(cleanName, extractPrefix) {
			continue
		}

		if err := extractZipFile(f, path.Join(extractPath, cleanName)); err != nil {
			return err
		}

		extracted++
	}

	if extracted == 0 {
		return ErrNoPublicFiles
	}

	return nil
}

func extractZipFile(f *zip.File, extractPath string) error {
	dir := filepath.Dir(extractPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// TODO handle symlinks

	r, err := f.Open()
	if err != nil {
		return err
	}
	defer r.Close()

	w, err := os.OpenFile(extractPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer w.Close()

	if _, err := io.Copy(w, r); err != nil {
		return err
	}

	return w.Close()
}
