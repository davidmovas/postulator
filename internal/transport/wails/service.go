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

type Deps struct {
	Sites     sitesUseCase
	Graph     graphUseCase
	Pages     pagesUseCase
	Templates templatesUseCase
	Runs      runsUseCase
	Sync      syncUseCase
	Reports   reportsUseCase
	Imports   importsUseCase
	Models    modelsUseCase
	Agent     agentUseCase
	Schedules schedulesUseCase
	Tools     toolCatalog
	Settings  SettingsDeps
}

func Services(logger *zap.Logger, build BuildInfo, deps Deps) []application.Service {
	return []application.Service{
		bind(NewHealthService(logger, build)),
		bind(NewSitesService(logger, deps.Sites)),
		bind(NewGraphService(logger, deps.Graph)),
		bind(NewPagesService(logger, deps.Pages)),
		bind(NewTemplatesService(logger, deps.Templates)),
		bind(NewRunsService(logger, deps.Runs)),
		bind(NewSyncService(logger, deps.Sync)),
		bind(NewReportsService(logger, deps.Reports)),
		bind(NewImportService(logger, deps.Imports)),
		bind(NewModelsService(logger, deps.Models)),
		bind(NewAgentService(logger, deps.Agent)),
		bind(NewSchedulesService(logger, deps.Schedules)),
		bind(NewToolsService(logger, deps.Tools)),
		bind(NewSettingsService(logger, deps.Settings)),
	}
}
