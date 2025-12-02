package app

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/config"
	"github.com/davidmovas/postulator/internal/domain"
	"github.com/davidmovas/postulator/internal/handlers"
	"github.com/davidmovas/postulator/internal/infra"
	"go.uber.org/fx"
)

type App struct {
	ctx            context.Context
	fxApp          *fx.App
	bindings       []any
	cfg            *config.Config
	dialogsHandler *handlers.DialogsHandler
}

func New(cfg *config.Config) (*App, error) {
	var (
		articlesHandler    *handlers.ArticlesHandler
		categoriesHandler  *handlers.CategoriesHandler
		jobsHandler        *handlers.JobsHandler
		promptsHandler     *handlers.PromptsHandler
		providersHandler   *handlers.ProvidersHandler
		sitesHandler       *handlers.SitesHandler
		healthCheckHandler *handlers.HealthCheckHandler
		statsHandler       *handlers.StatsHandler
		topicsHandler      *handlers.TopicsHandler
		importerHandler    *handlers.ImporterHandler
		settingsHandler    *handlers.SettingsHandler
		proxyHandler       *handlers.ProxyHandler
		mediaHandler       *handlers.MediaHandler
		dialogsHandler     *handlers.DialogsHandler
	)

	fxApp := fx.New(
		fx.Supply(cfg),

		infra.Module,
		domain.Module,
		handlers.Module,

		fx.Populate(
			&articlesHandler,
			&categoriesHandler,
			&jobsHandler,
			&promptsHandler,
			&providersHandler,
			&sitesHandler,
			&healthCheckHandler,
			&statsHandler,
			&topicsHandler,
			&importerHandler,
			&settingsHandler,
			&proxyHandler,
			&mediaHandler,
			&dialogsHandler,
		),
	)

	a := &App{
		fxApp: fxApp,
		bindings: []any{
			articlesHandler,
			categoriesHandler,
			jobsHandler,
			promptsHandler,
			providersHandler,
			sitesHandler,
			healthCheckHandler,
			statsHandler,
			topicsHandler,
			importerHandler,
			settingsHandler,
			proxyHandler,
			mediaHandler,
			dialogsHandler,
		},
		dialogsHandler: dialogsHandler,
		cfg: cfg,
	}

	return a, nil
}

func (a *App) Start(ctx context.Context) error {
	a.ctx = ctx
	return a.fxApp.Start(ctx)
}

// SetWailsContext sets the Wails runtime context for handlers that need it
func (a *App) SetWailsContext(ctx context.Context) {
	if a.dialogsHandler != nil {
		a.dialogsHandler.SetContext(ctx)
	}
}

func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return a.fxApp.Stop(ctx)
}

func (a *App) GetBinds() []any {
	return a.bindings
}
