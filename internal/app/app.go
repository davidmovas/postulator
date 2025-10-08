package app

import (
	"Postulator/internal/config"
	"Postulator/internal/infrastructure/database"
	"Postulator/pkg/di"
	"Postulator/pkg/logger"
	"fmt"
	"path/filepath"

	"github.com/adrg/xdg"
)

type App struct {
	container di.Container
	logger    *logger.Logger
	cfg       *config.Config
}

func New(cfg *config.Config) (*App, error) {
	c := di.New()

	c.MustRegister(di.Instance[*config.Config](cfg))

	if err := c.Register(
		di.For[*logger.Logger](func(c di.Container) (*logger.Logger, error) {
			l, err := logger.New(cfg)
			if err != nil {
				return nil, err
			}

			c.AddCloseFunc(func() {
				if err = l.Close(); err != nil {
					fmt.Println("Error while closing logger:", err)
				}
			})

			return l, nil
		}).AsSingleton(),

		di.For[*database.DB](func(c di.Container) (*database.DB, error) {
			path := filepath.Join(xdg.ConfigHome, "Postulator", "data.db")

			db, err := database.NewDB(path)
			if err != nil {
				return nil, err
			}

			c.AddCloseFunc(func() { db.Close() })
			return db, nil
		}).AsSingleton(),
	); err != nil {
		return nil, err
	}

	var l *logger.Logger
	if err := c.Resolve(&l); err != nil {
		return nil, err
	}

	return &App{
		container: c,
		logger:    l,
		cfg:       cfg,
	}, nil
}

func (a *App) Start() {
	a.logger.Info("Starting app")
}

func (a *App) Stop() {
	a.logger.Info("Stopping app")

	a.container.Close()
}
