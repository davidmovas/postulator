package infra

import (
	"context"

	"github.com/davidmovas/postulator/internal/infra/database"
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
	fx.Provide(database.NewDB),
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
)
