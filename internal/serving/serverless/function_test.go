package serverless

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFunctionHost(t *testing.T) {
	function := Function{
		Name:      "my-func",
		Domain:    "knative.example.com",
		Namespace: "my-namespace-123",
	}

	require.Equal(t, "my-func.my-namespace-123.knative.example.com", function.Host())
}
