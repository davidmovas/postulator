package ctx

import (
	"context"
	"time"
)

const (
	FastContextTimeout    = time.Second * 10
	MediumContextTimeout  = time.Second * 30
	LongContextTimeout    = time.Minute * 1
	AIContextTimeout      = time.Minute * 5
	ScannerContextTimeout = time.Minute * 10 // For scanning WordPress sites
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

// ScannerCtx returns a context with 10-minute timeout for scanning WordPress sites
func ScannerCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), ScannerContextTimeout)
	time.AfterFunc(ScannerContextTimeout, cancel)
	return ctx
}

// CancellableCtx returns a context with timeout and a cancel function
// Use this when you need to be able to cancel the operation externally
func CancellableCtx(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}
