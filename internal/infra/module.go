package infra

import (
	"context"

	"github.com/davidmovas/postulator/internal/config"
	"github.com/davidmovas/postulator/internal/infra/database"
	"github.com/davidmovas/postulator/internal/infra/importer"
	"github.com/davidmovas/postulator/internal/infra/secret"
	"github.com/davidmovas/postulator/internal/infra/wp"
	"github.com/davidmovas/postulator/pkg/logger"
	"go.uber.org/fx"
)

var Module = fx.Module("infra",
	// Logger
	fx.Provide(logger.New),
	fx.Invoke(func(lc fx.Lifecycle, log *logger.Logger) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				return log.Close()
			},
		})
	}),

	// Database
	fx.Provide(func(cfg *config.Config) (*database.DB, error) {
		return database.NewDB(cfg.DatabasePath)
	}),
	fx.Invoke(func(lc fx.Lifecycle, db *database.DB) {
		lc.Append(fx.Hook{
			OnStop: func(ctx context.Context) error {
				db.Close()
				return nil
			},
		})
	}),

	// Secret
	fx.Provide(secret.NewManager),

	// WordPress
	fx.Provide(wp.NewClient),

	// Importer service
	fx.Provide(importer.NewImportService),
)
