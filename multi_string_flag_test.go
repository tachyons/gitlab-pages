package main

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMultiStringFlagAppendsOnSet(t *testing.T) {
	var concrete MultiStringFlag
	var iface flag.Value

	iface = &concrete

	require.NoError(t, iface.Set("foo"))
	require.NoError(t, iface.Set("bar"))

	require.Equal(t, MultiStringFlag{"foo", "bar"}, concrete)
}
