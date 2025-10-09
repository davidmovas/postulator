package aiprovider

import (
	"Postulator/internal/domain/entities"
	"context"
)

type Repository interface {
	Create(ctx context.Context, provider *entities.AIProvider) error
	GetByID(ctx context.Context, id int64) (*entities.AIProvider, error)
	GetAll(ctx context.Context) ([]*entities.AIProvider, error)
	GetActive(ctx context.Context) ([]*entities.AIProvider, error)
	Update(ctx context.Context, provider *entities.AIProvider) error
	Delete(ctx context.Context, id int64) error
}
