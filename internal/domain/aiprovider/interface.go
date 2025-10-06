package aiprovider

import (
	"context"
)

type Repository interface {
	Create(ctx context.Context, provider *AIProvider) error
	GetByID(ctx context.Context, id int64) (*AIProvider, error)
	GetAll(ctx context.Context) ([]*AIProvider, error)
	GetActive(ctx context.Context) ([]*AIProvider, error)
	Update(ctx context.Context, provider *AIProvider) error
	Delete(ctx context.Context, id int64) error
}

type AIClient interface {
	GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}
