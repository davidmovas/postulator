package ai

import (
	"context"
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
