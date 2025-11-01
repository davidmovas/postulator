package providers

import (
	"context"
	"time"
)

type Type string

const (
	TypeOpenAI    Type = "openai"
	TypeAnthropic Type = "anthropic"
	TypeGoogle    Type = "google"
)

type Provider struct {
	ID        int64
	Name      string
	Type      Type
	APIKey    string
	Model     string
	IsActive  bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Model struct {
	ID         string
	Name       string
	Provider   Type
	MaxTokens  int
	InputCost  float64
	OutputCost float64
}

type Repository interface {
	Create(ctx context.Context, provider *Provider) error
	GetByID(ctx context.Context, id int64) (*Provider, error)
	GetAll(ctx context.Context) ([]*Provider, error)
	GetActive(ctx context.Context) ([]*Provider, error)
	Update(ctx context.Context, provider *Provider) error
	Delete(ctx context.Context, id int64) error
}

type Service interface {
	CreateProvider(ctx context.Context, provider *Provider) error
	GetProvider(ctx context.Context, id int64) (*Provider, error)
	ListProviders(ctx context.Context) ([]*Provider, error)
	ListActiveProviders(ctx context.Context) ([]*Provider, error)
	UpdateProvider(ctx context.Context, provider *Provider) error
	DeleteProvider(ctx context.Context, id int64) error
	SetProviderStatus(ctx context.Context, id int64, isActive bool) error

	GetAvailableModels(providerType Type) ([]*Model, error)
	ValidateModel(providerType Type, model string) error
}
