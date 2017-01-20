package main

import (
	"flag"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestMultiStringFlagAppendsOnSet(t *testing.T) {
	var concrete MultiStringFlag
	var iface flag.Value

	iface = &concrete

	assert.NoError(t, iface.Set("foo"))
	assert.NoError(t, iface.Set("bar"))

	assert.Equal(t, MultiStringFlag{"foo", "bar"}, concrete)
}
