package request

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWithHTTPSFlag(t *testing.T) {
	r, err := http.NewRequest("GET", "/", nil)
	require.NoError(t, err)

	assert.Panics(t, func() {
		IsHTTPS(r)
	})

	httpsRequest := WithHTTPSFlag(r, true)
	assert.True(t, IsHTTPS(httpsRequest))

	httpRequest := WithHTTPSFlag(r, false)
	assert.False(t, IsHTTPS(httpRequest))
}
