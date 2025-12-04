package settings

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value string) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (bool, error)
}

type Service interface {
	GetHealthCheckSettings(ctx context.Context) (*entities.HealthCheckSettings, error)
	UpdateHealthCheckSettings(ctx context.Context, settings *entities.HealthCheckSettings) error
	GetProxySettings(ctx context.Context) (*entities.ProxySettings, error)
	UpdateProxySettings(ctx context.Context, settings *entities.ProxySettings) error
	GetDashboardSettings(ctx context.Context) (*entities.DashboardSettings, error)
	UpdateDashboardSettings(ctx context.Context, settings *entities.DashboardSettings) error
}

type HealthCheckScheduler interface {
	ApplySettings(ctx context.Context, enabled bool, intervalMinutes int) error
}
