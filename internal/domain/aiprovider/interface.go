package aiprovider

import (
	"Postulator/internal/domain/entities"
	"context"
)

type IRepository interface {
	Create(ctx context.Context, provider *entities.AIProvider) error
	GetByID(ctx context.Context, id int64) (*entities.AIProvider, error)
	GetAll(ctx context.Context) ([]*entities.AIProvider, error)
	GetActive(ctx context.Context) ([]*entities.AIProvider, error)
	Update(ctx context.Context, provider *entities.AIProvider) error
	Delete(ctx context.Context, id int64) error
}

type IService interface {
	CreateProvider(ctx context.Context, provider *entities.AIProvider) error
	GetProvider(ctx context.Context, id int64) (*entities.AIProvider, error)
	ListProviders(ctx context.Context) ([]*entities.AIProvider, error)
	ListActiveProviders(ctx context.Context) ([]*entities.AIProvider, error)
	UpdateProvider(ctx context.Context, provider *entities.AIProvider) error
	DeleteProvider(ctx context.Context, id int64) error
	SetProviderStatus(ctx context.Context, id int64, isActive bool) error

	// Model-related methods

	GetAvailableModels(providerName string) []entities.AIModel
	ValidateModel(providerName string, model string) error
}
