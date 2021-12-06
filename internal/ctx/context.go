package ctx

import (
	"context"
	"time"
)

type internalCtx string

const (
	startTimeCtxKey  internalCtx = "start_time_ctx"
	lookupPathCtxKey internalCtx = "lookup_path_ctx"
)

// SetStartTime in context
func SetStartTime(ctx context.Context, start time.Time) context.Context {
	return context.WithValue(ctx, startTimeCtxKey, start)
}

// GetStartTime from context
func GetStartTime(ctx context.Context) time.Time {
	value := ctx.Value(startTimeCtxKey)

	startTime, ok := value.(time.Time)
	if !ok {
		return time.Time{}
	}

	return startTime
}
