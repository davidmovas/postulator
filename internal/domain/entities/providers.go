package entities

import "time"

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
	ID              string
	Name            string
	Provider        Type
	ContextWindow   int     // Max input tokens (context window)
	MaxOutputTokens int     // Max output tokens
	InputCost       float64 // Cost per 1M input tokens
	OutputCost      float64 // Cost per 1M output tokens
	RPM             int     // Requests per minute
	TPM             int     // Tokens per minute
	// UsesCompletionTokens indicates if the model uses max_completion_tokens instead of max_tokens
	UsesCompletionTokens bool
	// IsReasoningModel indicates models that don't support temperature (o1, o3, gpt-5 series)
	IsReasoningModel bool
	// SupportsWebSearch indicates if the model supports web search tool
	SupportsWebSearch bool
}
