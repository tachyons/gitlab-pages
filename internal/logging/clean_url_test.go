package logging

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRemovingAllSensitiveData(t *testing.T) {
	url := CleanURL("https://user:password@gitlab.com/gitlab?key=value#fragment")
	require.Equal(t, "https://gitlab.com/gitlab", url)
}

func TestInvalidURL(t *testing.T) {
	require.Empty(t, CleanURL("://invalid URL"))
}
