package wails

import (
	"context"

	"github.com/wailsapp/wails/v3/pkg/application"
	"go.uber.org/zap"

	kernelctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/middleware"
)

func Wrap[In, Out any](logger *zap.Logger, operation string, next middleware.Handler[In, Out]) middleware.Handler[In, Out] {
	audited := middleware.Audit(logger, operation, middleware.Recover(next))

	return func(c context.Context, in In) (Out, error) {
		out, err := audited(kernelctx.WithActor(c, kernelctx.ActorUser), in)
		if err != nil {
			var zero Out
			return zero, Convert(err)
		}
		return out, nil
	}
}

func bind[T any](instance *T) application.Service {
	return application.NewServiceWithOptions(instance, application.ServiceOptions{MarshalError: MarshalError})
}

func Services(logger *zap.Logger, build BuildInfo) []application.Service {
	return []application.Service{
		bind(NewHealthService(logger, build)),
	}
}
