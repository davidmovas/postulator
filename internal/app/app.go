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
	ctx      context.Context
	fxApp    *fx.App
	bindings []any
	cfg      *config.Config
}

func New(cfg *config.Config) (*App, error) {
	var (
		articlesHandler   *handlers.ArticlesHandler
		categoriesHandler *handlers.CategoriesHandler
		jobsHandler       *handlers.JobsHandler
		promptsHandler    *handlers.PromptsHandler
		providersHandler  *handlers.ProvidersHandler
		sitesHandler      *handlers.SitesHandler
		statsHandler      *handlers.StatsHandler
		topicsHandler     *handlers.TopicsHandler
		settingsHandler   *handlers.SettingsHandler
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
			&statsHandler,
			&topicsHandler,
			&settingsHandler,
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
			statsHandler,
			topicsHandler,
			settingsHandler,
		},
		cfg: cfg,
	}

	return a, nil
}

func (a *App) Start(ctx context.Context) error {
	a.ctx = ctx
	return a.fxApp.Start(ctx)
}

func (a *App) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return a.fxApp.Stop(ctx)
}

func (a *App) GetBinds() []any {
	return a.bindings
}
