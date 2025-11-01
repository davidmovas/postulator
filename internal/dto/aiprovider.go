package dto

import "github.com/davidmovas/postulator/internal/domain/entities"

type AIProvider struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Provider  string `json:"provider"`
	Model     string `json:"model"`
	IsActive  bool   `json:"isActive"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt"`
}

func FromAIProvider(e *entities.AIProvider) *AIProvider {
	if e == nil {
		return nil
	}
	return &AIProvider{
		ID:        e.ID,
		Name:      e.Name,
		Provider:  e.Provider,
		Model:     e.Model,
		IsActive:  e.IsActive,
		CreatedAt: e.CreatedAt.UTC().Format(timeLayout),
		UpdatedAt: e.UpdatedAt.UTC().Format(timeLayout),
	}
}

func FromAIProviders(items []*entities.AIProvider) []*AIProvider {
	out := make([]*AIProvider, 0, len(items))
	for _, it := range items {
		out = append(out, FromAIProvider(it))
	}
	return out
}

// Expose model constants for frontend selections

type ModelsByProvider struct {
	OpenAI    []string `json:"openai"`
	Anthropic []string `json:"anthropic"`
	Google    []string `json:"google"`
}

func GetAllModels() *ModelsByProvider {
	toStr := func(models []entities.AIModel) []string {
		out := make([]string, 0, len(models))
		for _, m := range models {
			out = append(out, string(m))
		}
		return out
	}
	return &ModelsByProvider{
		OpenAI:    toStr(entities.GetModelsByProvider(entities.ProviderOpenAI)),
		Anthropic: toStr(entities.GetModelsByProvider(entities.ProviderAnthropic)),
		Google:    toStr(entities.GetModelsByProvider(entities.ProviderGoogle)),
	}
}
