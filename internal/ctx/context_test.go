package ctx

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSetAndGetStartTime(t *testing.T) {
	start := time.Now()
	ctx := context.Background()
	ctx = SetStartTime(ctx, start)

	timeFromCtx := GetStartTime(ctx)
	require.Equal(t, start, timeFromCtx)
}
