package ai

import "context"

type Client interface {
	GenerateArticle(ctx context.Context, systemPrompt, userPrompt string) (string, error)
	GenerateTopicVariation(ctx context.Context, topic string, amount int) ([]string, error)
}
