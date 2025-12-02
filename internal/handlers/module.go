package handlers

import "go.uber.org/fx"

var Module = fx.Module("handlers",
	fx.Provide(
		NewArticlesHandler,
		NewCategoriesHandler,
		NewJobsHandler,
		NewPromptsHandler,
		NewProvidersHandler,
		NewSitesHandler,
		NewStatsHandler,
		NewTopicsHandler,
		NewSettingsHandler,
		NewHealthCheckHandler,
		NewImporterHandler,
		NewProxyHandler,
		NewMediaHandler,
		NewDialogsHandler,
		NewAppHandler,
	),
)
