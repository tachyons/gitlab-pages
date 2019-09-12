package host

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFromString(t *testing.T) {
	require.Equal(t, "example.com", FromString("example.com"))
	require.Equal(t, "example.com", FromString("eXAmpLe.com"))
	require.Equal(t, "example.com", FromString("example.com:8080"))
}

func TestFromRequest(t *testing.T) {
	require.Equal(t, "example.com", FromRequest(httptest.NewRequest("GET", "example.com:8080/123", nil)))
}
