package app

import "Postulator/internal/config"

type App struct {
	cfg *config.Config
}

func New(cfg *config.Config) (*App, error) {

	return &App{
		cfg: cfg,
	}, nil
}

func (a *App) Start() {}

func (a *App) Stop() {}
