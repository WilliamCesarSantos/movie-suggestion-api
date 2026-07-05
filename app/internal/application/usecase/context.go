package usecase

import "context"

// detachedContext wraps a parent context, preserving its values but ignoring its cancellation.
// This is useful for background goroutines that need context values (e.g. correlationId, logger)
// but should not be cancelled when the parent request context is done.
type detachedContext struct {
	context.Context
	values context.Context
}

func (d *detachedContext) Value(key any) any {
	return d.values.Value(key)
}

// detachContext returns a new context that preserves the values from parent but is never cancelled.
func detachContext(parent context.Context) context.Context {
	return &detachedContext{Context: context.Background(), values: parent}
}
