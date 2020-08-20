package zip

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const URL = "http://192.168.88.233:9000/test-bucket/doc-gitlab-com.zip?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=TEST_KEY%2F20200820%2F%2Fs3%2Faws4_request&X-Amz-Date=20200820T152420Z&X-Amz-Expires=432000&X-Amz-SignedHeaders=host&X-Amz-Signature=fcf49604f53564ce1648e5a0c2d8f1186ba3d9dd5e40d2c3244c57053e0348e9"

func TestOpenArchive(t *testing.T) {
	zip := newArchive(URL)
	defer zip.close()

	println("OpenArchive")
	ts := time.Now()
	err := zip.openArchive(context.Background())
	println(time.Since(ts).String())
	require.NoError(t, err)

	println("Open")
	ts = time.Now()
	rc, err := zip.Open(context.Background(), "sitemap.xml")
	println(time.Since(ts).String())
	require.NoError(t, err)

	println("ReadAll")
	ts = time.Now()
	data, err := ioutil.ReadAll(rc)
	println(time.Since(ts).String())
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Contains(t, string(data), "<url>")
}
