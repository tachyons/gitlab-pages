package disk

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenNoFollow(t *testing.T) {
	tmpfile, err := ioutil.TempFile("", "link-test")
	require.NoError(t, err)
	defer tmpfile.Close()

	orig := tmpfile.Name()
	softLink := orig + ".link"
	defer os.Remove(orig)

	source, err := openNoFollow(orig)
	require.NoError(t, err)
	require.NotNil(t, source)
	defer source.Close()

	err = os.Symlink(orig, softLink)
	require.NoError(t, err)
	defer os.Remove(softLink)

	link, err := openNoFollow(softLink)
	require.Error(t, err)
	require.Nil(t, link)
}
