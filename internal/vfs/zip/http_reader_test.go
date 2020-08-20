package zip

import (
	"archive/zip"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/require"
)

//const URL = "https://storage.googleapis.com/gitlab-gprd-artifacts/5b/f5/5bf596ed115cd2bf53b9ac39f77563ee26275e449f62c0b96a0fdb5d719ac6da/2020_08_18/692126387/760066867/artifacts.zip?response-content-disposition=attachment%3B%20filename%3D%22artifacts.zip%22%3B%20filename%2A%3DUTF-8%27%27artifacts.zip&response-content-type=application%2Fzip&GoogleAccessId=gitlab-object-storage-prd@gitlab-production.iam.gserviceaccount.com&Signature=2yWByo4dT4Ic6fkrm2asb0TinnFHnELcciZ6qB6Nhc8eNyTYxaZe9aOQblwR%0AWz9yiI84zaD%2F0eZtiJI06OqGz3u%2Bchsc7Mn%2BEdthhmcR9lIUJrbQh96BUEJf%0A0GniiYOGhEdr2gK9sr%2FYPiX7jv4ABkMmyr%2BZdxCPd1%2F%2FnIGFVyTdX07CbsrI%0AYA67RGOez1w9RqcF0wAy5qKs57E9fXshc%2BWqRZD%2Fbtd4PysOHGAT47i7Vslt%0AuzcaGXyiN7Hl8Ckq3WimeacCoB9L%2FNntsLcwx3llKdE0gpzAH04vjiVi705p%0AKV16FuRsF4qbhjYcKjzUao33QVGGXpdOsikF7JrnmA%3D%3D&Expires=1597782702"
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
