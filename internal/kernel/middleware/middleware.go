package middleware

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	kernelctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type Handler[In, Out any] func(context.Context, In) (Out, error)

func Recover[In, Out any](next Handler[In, Out]) Handler[In, Out] {
	return func(c context.Context, in In) (out Out, err error) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			var zero Out
			out = zero
			err = errors.New(errors.Internal, "handler panicked").WithDetail("panic", describe(recovered))
		}()
		return next(c, in)
	}
}

func describe(recovered any) string {
	if asError, ok := recovered.(error); ok {
		return asError.Error()
	}
	return fmt.Sprint(recovered)
}

func Audit[In, Out any](logger *zap.Logger, operation string, next Handler[In, Out]) Handler[In, Out] {
	return func(c context.Context, in In) (Out, error) {
		started := time.Now()
		out, err := next(c, in)

		fields := []zap.Field{
			zap.String("operation", operation),
			zap.Int64("durationMs", time.Since(started).Milliseconds()),
		}
		if actor, ok := kernelctx.ActorFrom(c); ok {
			fields = append(fields, zap.String("actor", actor.String()))
		}

		if err != nil {
			fields = append(fields, zap.String("code", errors.CodeOf(err).String()), zap.Error(err))
			logger.Warn("operation failed", fields...)
			return out, err
		}

		logger.Info("operation completed", fields...)
		return out, nil
	}
}
