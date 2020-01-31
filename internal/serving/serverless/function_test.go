package serverless

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFunctionHost(t *testing.T) {
	function := Function{
		Name:       "my-func",
		Namespace:  "my-namespace-123",
		BaseDomain: "knative.example.com",
	}

	require.Equal(t, "my-func.my-namespace-123.knative.example.com", function.Host())
}
