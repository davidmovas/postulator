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
	ID         string
	Name       string
	Provider   Type
	MaxTokens  int
	InputCost  float64
	OutputCost float64
}
