package app

import (
	"Postulator/internal/config"
	"Postulator/pkg/di"
)

type App struct {
	container di.Container
	cfg       *config.Config
}

func New(cfg *config.Config) (*App, error) {
	c := di.New()

	return &App{
		container: c,
		cfg:       cfg,
	}, nil
}

func (a *App) Start() {}

func (a *App) Stop() {
	a.container.Close()
}
