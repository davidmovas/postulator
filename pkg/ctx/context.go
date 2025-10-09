package ctx

import (
	"context"
	"time"
)

const (
	FastContextTimeout = time.Second * 5
)

func FastCtx() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), FastContextTimeout)
	time.AfterFunc(FastContextTimeout, cancel)
	return ctx
}
