package app

import (
	"context"
	"time"

	"github.com/davidmovas/postulator/internal/config"
	"github.com/davidmovas/postulator/internal/domain"
	"github.com/davidmovas/postulator/internal/infra"
	"github.com/davidmovas/postulator/pkg/logger"
	"go.uber.org/fx"
)

type App struct {
	ctx    context.Context
	fxApp  *fx.App
	logger *logger.Logger
	cfg    *config.Config
}

func New(cfg *config.Config) (*App, error) {
	fxApp := fx.New(
		fx.Supply(cfg),
		infra.Module,
		domain.Module,
	)

	a := &App{
		fxApp: fxApp,
		cfg:   cfg,
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
