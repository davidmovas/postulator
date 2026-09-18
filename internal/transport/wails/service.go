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

type Source[T any] func() (T, error)

func call[T, In, Out any](source Source[T], method func(T, context.Context, In) (Out, error)) middleware.Handler[In, Out] {
	return func(c context.Context, in In) (Out, error) {
		live, err := source()
		if err != nil {
			var zero Out
			return zero, err
		}
		return method(live, c, in)
	}
}

func bind[T any](instance *T) application.Service {
	return application.NewServiceWithOptions(instance, application.ServiceOptions{MarshalError: MarshalError})
}

type Deps struct {
	Sites     Source[SitesUseCase]
	Graph     Source[GraphUseCase]
	Pages     Source[PagesUseCase]
	Templates Source[TemplatesUseCase]
	Runs      Source[RunsUseCase]
	Sync      Source[SyncUseCase]
	Reports   Source[ReportsUseCase]
	Imports   Source[ImportsUseCase]
	Models    Source[ModelsUseCase]
	Agent     Source[AgentUseCase]
	Schedules Source[SchedulesUseCase]
	Tools     Source[ToolCatalog]
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
