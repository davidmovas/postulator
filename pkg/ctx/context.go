package ctx

import (
	"context"
	"time"
)

const (
	FastContextTimeout   = time.Second * 5
	MediumContextTimeout = time.Second * 10
	LongContextTimeout   = time.Second * 30
	AIContextTimeout     = time.Minute * 5
)

func FastCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), FastContextTimeout)
	time.AfterFunc(FastContextTimeout, cancel)
	return ctx
}

func MediumCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), MediumContextTimeout)
	time.AfterFunc(MediumContextTimeout, cancel)
	return ctx
}

func LongCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), LongContextTimeout)
	time.AfterFunc(LongContextTimeout, cancel)
	return ctx
}

func WithTimeout(timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	time.AfterFunc(timeout, cancel)
	return ctx
}

// AICtx returns a context with 5-minute timeout for AI operations
func AICtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), AIContextTimeout)
	time.AfterFunc(AIContextTimeout, cancel)
	return ctx
}

// CancellableCtx returns a context with timeout and a cancel function
// Use this when you need to be able to cancel the operation externally
func CancellableCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
