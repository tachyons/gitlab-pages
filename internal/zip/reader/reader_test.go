package reader

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewReader(t *testing.T) {
	zipFile, err := os.Open("../../../shared/pages/group/zip.gitlab.io/public.zip")
	require.NoError(t, err)
	zfi, err := zipFile.Stat()
	require.NoError(t, err)

	reader, err := New(zipFile, zfi.Size())
	require.NoError(t, err)

	f, fi, err := reader.Open("index.html")
	require.NoError(t, err)
	defer f.Close()
	require.NotZero(t, fi.Size())

	actualContents, err := ioutil.ReadAll(f)
	require.NoError(t, err, "read zip entry contents")
	require.Equal(t, "zip/index.html\n", string(actualContents), "compare zip entry contents")

	_, _, err = reader.Open("unknown.html")
	require.EqualError(t, err, "\"public/unknown.html\": not found")
}
