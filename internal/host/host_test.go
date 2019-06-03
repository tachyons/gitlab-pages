package host

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFromString(t *testing.T) {
	assert.Equal(t, "example.com", FromString("example.com"))
	assert.Equal(t, "example.com", FromString("eXAmpLe.com"))
	assert.Equal(t, "example.com", FromString("example.com:8080"))
}

func TestFromRequest(t *testing.T) {
	assert.Equal(t, "example.com", FromRequest(httptest.NewRequest("GET", "example.com:8080/123", nil)))
}
