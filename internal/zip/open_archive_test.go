package zip

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenHTTPArchive(t *testing.T) {
	const (
		zipFile   = "public.zip"
		entryName = "public/index.html"
		contents  = "zip/index.html\n"
		testRoot  = "../../shared/pages/group/zip.gitlab.io/"
	)

	srv := httptest.NewServer(http.FileServer(http.Dir(testRoot)))
	defer srv.Close()

	zr, err := OpenArchive(context.Background(), srv.URL+"/"+zipFile)
	require.NoError(t, err, "call OpenArchive")
	require.Len(t, zr.Archive().File, 2)

	zf := zr.Archive().File[1]
	require.Equal(t, entryName, zf.Name, "zip entry name")

	entry, err := zf.Open()
	require.NoError(t, err, "get zip entry reader")
	defer entry.Close()

	actualContents, err := ioutil.ReadAll(entry)
	require.NoError(t, err, "read zip entry contents")
	require.Equal(t, contents, string(actualContents), "compare zip entry contents")
}

func TestOpenFileArchive(t *testing.T) {
	const (
		entryName = "public/index.html"
		contents  = "zip/index.html\n"
		testRoot  = "../../shared/pages/group/zip.gitlab.io/public.zip"
	)
	zr, err := OpenArchive(context.Background(), testRoot)
	require.NoError(t, err, "call OpenArchive")
	require.Len(t, zr.Archive().File, 2)

	zf := zr.Archive().File[1]
	require.Equal(t, entryName, zf.Name, "zip entry name")

	entry, err := zf.Open()
	require.NoError(t, err, "get zip entry reader")
	defer entry.Close()

	actualContents, err := ioutil.ReadAll(entry)
	require.NoError(t, err, "read zip entry contents")
	require.Equal(t, contents, string(actualContents), "compare zip entry contents")
}
