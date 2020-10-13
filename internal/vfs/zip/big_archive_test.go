package zip

import (
	"archive/zip"
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"gitlab.com/gitlab-org/gitlab-pages/internal/testhelpers"
)

func TestBigArchive(t *testing.T) {
	archive, cleanup := openZipArchiveCustom(t, "../../tmp/docs.zip", nil)
	defer cleanup()

	logAll = true
	t.Log("BaseFile", sizeOf(zip.File{}, nil))
	logAll = false

	t.Log("FileCount:", archive.FileCount())
	t.Log("Size:", archive.Size())
	t.Log("SizePerFile:", archive.SizePerFile())
	t.Log("ZipSize:", archive.ZipSize())
	t.Log("ZipSizePerFile:", archive.ZipSizePerFile())
}

func TestString(t *testing.T) {
	value := reflect.ValueOf("test")
	t.Log("kind", value.Kind())
	t.Log("pointer", value.Pointer())
}

type zipFiles []*zip.File

func (p zipFiles) Len() int           { return len(p) }
func (p zipFiles) Less(i, j int) bool { return p[i].Name < p[j].Name }
func (p zipFiles) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

const hashMod = 32

func hash(s string) uint32 {
	var hash uint32
	hash = 2166136261
	for _, c := range s {
		hash *= 16777619
		hash ^= uint32(c)
	}
	return hash % hashMod
}

type zipFilePtr struct {
	*zip.File
}

func BenchmarkTestAccess(t *testing.B) {
	archive, cleanup := openZipArchiveCustomB(t, "../../tmp/docs.zip", nil)
	defer cleanup()

	var lastMapPtr map[string]*zip.File
	var lastMapStruct map[string]zip.File
	var lastMapStructPtr map[string]zipFilePtr
	var lastMapCopyPtr map[string]*zip.File
	var lastMapHashedPtr []map[string]*zip.File
	var lastMapCopyFlatPtr map[string]*zip.File
	var lastSlice zipFiles

	t.Run("create Map Ptr", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			lastMapPtr = make(map[string]*zip.File)

			for _, file := range archive.files {
				lastMapPtr[file.Name] = file
			}
		}
	})

	t.Run("create Map Struct", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			lastMapStruct = make(map[string]zip.File)

			for _, file := range archive.files {
				lastMapStruct[file.Name] = *file
			}
		}
	})

	t.Run("create Map StructPtr", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			lastMapStructPtr = make(map[string]zipFilePtr)

			for _, file := range archive.files {
				lastMapStructPtr[file.Name] = zipFilePtr{File: file}
			}
		}
	})

	t.Run("create Map Hashed", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			lastMapHashedPtr = make([]map[string]*zip.File, hashMod)

			for i := 0; i < hashMod; i++ {
				lastMapHashedPtr[i] = make(map[string]*zip.File)
			}

			for _, file := range archive.files {
				lastMapHashedPtr[hash(file.Name)][file.Name] = file
			}
		}
	})

	t.Run("create Map Copy Pointer", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			lastMapCopyPtr = make(map[string]*zip.File)

			for _, file := range archive.files {
				newFile := &zip.File{}
				*newFile = *file
				lastMapCopyPtr[file.Name] = newFile
			}
		}
	})

	t.Run("create Map Copy Flat Pointer", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			ptrs := make([]zip.File, len(archive.files))
			lastMapCopyFlatPtr = make(map[string]*zip.File)

			for _, file := range archive.files {
				newFile := &ptrs[len(lastMapCopyFlatPtr)]
				*newFile = *file
				lastMapCopyFlatPtr[file.Name] = newFile
			}
		}
	})

	t.Run("create slice", func(t *testing.B) {
		for i := 0; i < t.N; i++ {
			lastSlice = make(zipFiles, len(archive.files))
			idx := 0

			for _, file := range archive.files {
				lastSlice[idx] = file
				idx++
			}

			sort.Sort(lastSlice)
		}
	})

	tests := []string{
		lastSlice[0].Name,
		lastSlice[len(lastSlice)/2].Name,
		lastSlice[len(lastSlice)-1].Name,
		"not/existing",
	}

	for _, test := range tests {
		t.Run("file: "+test, func(t *testing.B) {
			t.Run("map ptr", func(t *testing.B) {
				for i := 0; i < t.N; i++ {
					_ = lastMapPtr[test]
				}
			})

			t.Run("map struct", func(t *testing.B) {
				for i := 0; i < t.N; i++ {
					_ = lastMapStruct[test]
				}
			})

			t.Run("map struct ptr", func(t *testing.B) {
				for i := 0; i < t.N; i++ {
					_ = lastMapStructPtr[test]
				}
			})

			t.Run("map copy ptr", func(t *testing.B) {
				for i := 0; i < t.N; i++ {
					_ = lastMapCopyPtr[test]
				}
			})

			t.Run("map hashed ptr", func(t *testing.B) {
				for i := 0; i < t.N; i++ {
					_ = lastMapHashedPtr[hash(test)][test]
				}
			})

			t.Run("binary search", func(t *testing.B) {
				for i := 0; i < t.N; i++ {
					idx := sort.Search(len(lastSlice), func(i int) bool { return lastSlice[i].Name >= test })
					if idx >= 0 {
						item := lastSlice[idx]
						if item.Name == test {
							// no-op
						}
					}
				}
			})
		})
	}
}

func openZipArchiveCustomB(t *testing.B, path string, requests *int64) (*zipArchive, func()) {
	t.Helper()

	if requests == nil {
		requests = new(int64)
	}

	testServerURL, cleanup := newZipFileServerURLB(t, path, requests)

	fs := New().(*zipVFS)
	zip := newArchive(fs, testServerURL+"/public.zip", time.Second)

	err := zip.openArchive(context.Background())
	require.NoError(t, err)

	// public/ public/index.html public/404.html public/symlink.html
	// public/subdir/ public/subdir/hello.html public/subdir/linked.html
	// public/bad_symlink.html public/subdir/2bp3Qzs...
	require.NotZero(t, zip.files)
	require.Equal(t, int64(3), atomic.LoadInt64(requests), "we expect three requests to open ZIP archive: size and two to seek central directory")

	return zip, func() {
		cleanup()
	}
}

func newZipFileServerURLB(t *testing.B, zipFilePath string, requests *int64) (string, func()) {
	t.Helper()

	chdir := testhelpers.ChdirInPath(t, "../../../shared/pages", &chdirSet)

	m := http.NewServeMux()
	m.HandleFunc("/public.zip", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, zipFilePath)
		if requests != nil {
			atomic.AddInt64(requests, 1)
		}
	}))

	testServer := httptest.NewServer(m)

	return testServer.URL, func() {
		chdir()
		testServer.Close()
	}
}
