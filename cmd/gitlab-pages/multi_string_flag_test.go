package main

import (
	"flag"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiStringFlagAppendsOnSet(t *testing.T) {
	var concrete MultiStringFlag
	var iface flag.Value

	iface = &concrete

	assert.NoError(t, iface.Set("foo"))
	assert.NoError(t, iface.Set("bar"))

	assert.Equal(t, MultiStringFlag{"foo", "bar"}, concrete)
}
