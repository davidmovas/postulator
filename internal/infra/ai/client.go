package ai

import (
	"context"

	"github.com/davidmovas/postulator/internal/domain/entities"
)

type Client interface {
	GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (*ArticleResult, error)
	GenerateTopicVariations(ctx context.Context, topic string, amount int) ([]string, error)
}

type ArticleResult struct {
	Title      string
	Content    string
	TokensUsed int
	Cost       float64
}

// Factory создаёт AI клиентов
type Factory interface {
	CreateClient(provider *entities.Provider) (Client, error)
	GetAvailableModels(providerType entities.Type) []*entities.Model
	ValidateModel(providerType entities.Type, model string) bool
}
