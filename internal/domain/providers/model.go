package providers

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Repository interface {
	Create(ctx context.Context, provider *entities.Provider) error
	GetByID(ctx context.Context, id int64) (*entities.Provider, error)
	GetAll(ctx context.Context) ([]*entities.Provider, error)
	GetActive(ctx context.Context) ([]*entities.Provider, error)
	Update(ctx context.Context, provider *entities.Provider) error
	Delete(ctx context.Context, id int64) error
}

type Service interface {
	CreateProvider(ctx context.Context, provider *entities.Provider) error
	GetProvider(ctx context.Context, id int64) (*entities.Provider, error)
	ListProviders(ctx context.Context) ([]*entities.Provider, error)
	ListActiveProviders(ctx context.Context) ([]*entities.Provider, error)
	UpdateProvider(ctx context.Context, provider *entities.Provider) error
	DeleteProvider(ctx context.Context, id int64) error
	SetProviderStatus(ctx context.Context, id int64, isActive bool) error

	GetAvailableModels(providerType entities.Type) ([]*entities.Model, error)
	ValidateModel(providerType entities.Type, model string) error
}
