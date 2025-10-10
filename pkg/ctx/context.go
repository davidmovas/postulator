package ctx

import (
	"context"
	"time"
)

const (
	FastContextTimeout   = time.Second * 5
	MediumContextTimeout = time.Second * 10
	LongContextTimeout   = time.Second * 30
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
