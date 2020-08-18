package zip

import (
	"archive/zip"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

const URL = "http://192.168.88.233:9000/test-bucket/doc-gitlab-com.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=TEST_KEY%2F20200818%2F%2Fs3%2Faws4_request&X-Amz-Date=20200818T173935Z&X-Amz-Expires=432000&X-Amz-SignedHeaders=host&X-Amz-Signature=95810918d1b2441a07385838ebba5a0f01fdf4dcdf94ea9c602f8e7d06c84019"

func findZipFile(zip *zip.Reader, name string) *zip.File {
	for _, file := range zip.File {
		if file.Name == name {
			return file
		}
	}
	return nil
}

func TestOpenArchiveHTTP(t *testing.T) {
	zip, closer, err := openZIPHTTPArchive(URL)
	require.NoError(t, err)
	defer closer.Close()

	require.NotNil(t, zip)
	require.NotEmpty(t, zip.File)

	sitemap := findZipFile(zip, "public/sitemap.xml")
	require.NotNil(t, sitemap)

	println("DataOffset")
	_, err = sitemap.DataOffset()
	require.NoError(t, err)

	println("Open")
	rc, err := sitemap.Open()
	require.NoError(t, err)
	defer rc.Close()

	println("ReadAll")
	data, err := ioutil.ReadAll(rc)
	require.NoError(t, err)
	require.NotNil(t, data)
}
