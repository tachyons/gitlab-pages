package zip

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

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
