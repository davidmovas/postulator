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

func Timeout[In, Out any](limit time.Duration, next Handler[In, Out]) Handler[In, Out] {
	return func(c context.Context, in In) (Out, error) {
		type outcome struct {
			out       Out
			err       error
			recovered any
			panicked  bool
		}

		c, cancel := context.WithTimeout(c, limit)
		defer cancel()

		done := make(chan outcome, 1)
		go func() {
			defer func() {
				if recovered := recover(); recovered != nil {
					done <- outcome{panicked: true, recovered: recovered}
				}
			}()
			out, err := next(c, in)
			done <- outcome{out: out, err: err}
		}()

		select {
		case result := <-done:
			if result.panicked {
				panic(result.recovered)
			}
			return result.out, result.err
		case <-c.Done():
			var zero Out
			return zero, errors.Wrap(c.Err(), errors.Cancelled, "operation did not finish in time")
		}
	}
}

func Audit[In, Out any](logger *zap.Logger, operation string, next Handler[In, Out]) Handler[In, Out] {
	return func(c context.Context, in In) (Out, error) {
		started := time.Now()
		out, err := next(c, in)

		fields := []zap.Field{
			zap.String("operation", operation),
			zap.Int64("durationMs", time.Since(started).Milliseconds()),
		}
		if runID, ok := kernelctx.RunID(c); ok {
			fields = append(fields, zap.String("runId", runID))
		}
		if conversationID, ok := kernelctx.ConversationID(c); ok {
			fields = append(fields, zap.String("conversationId", conversationID))
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
