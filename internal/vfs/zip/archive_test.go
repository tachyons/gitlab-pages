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

	ts := time.Now()
	println("OpenArchive")
	err := zip.openArchive(context.Background())
	require.NoError(t, err)

	println("Open")
	rc, err := zip.Open(context.Background(), "public/sitemap.xml")
	require.NoError(t, err)

	println("ReadAll")
	data, err := ioutil.ReadAll(rc)
	require.NoError(t, err)
	require.NotNil(t, data)
	require.Contains(t, string(data), "<url>")
}
