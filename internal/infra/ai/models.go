package ai

import "github.com/davidmovas/postulator/internal/domain/entities"

var availableModels = map[entities.Type][]*entities.Model{
	entities.TypeOpenAI: {
		{
			ID:         "gpt-4o-mini",
			Name:       "GPT-4o Mini",
			Provider:   entities.TypeOpenAI,
			MaxTokens:  128000,
			InputCost:  0.15,
			OutputCost: 0.60,
		},
		{
			ID:         "gpt-5",
			Name:       "GPT-5",
			Provider:   entities.TypeOpenAI,
			MaxTokens:  128000,
			InputCost:  1.25,
			OutputCost: 10.00,
		},
		{
			ID:         "gpt-5-mini",
			Name:       "GPT-5 Mini",
			Provider:   entities.TypeOpenAI,
			MaxTokens:  128000,
			InputCost:  0.25,
			OutputCost: 2.00,
		},
		{
			ID:         "gpt-5-nano",
			Name:       "GPT-5 Nano",
			Provider:   entities.TypeOpenAI,
			MaxTokens:  128000,
			InputCost:  0.05,
			OutputCost: 0.40,
		},
		{
			ID:         "gpt-4.1",
			Name:       "GPT-4.1",
			Provider:   entities.TypeOpenAI,
			MaxTokens:  32768,
			InputCost:  2.00,
			OutputCost: 8.00,
		},
		{
			ID:         "gpt-4.1-nano",
			Name:       "GPT-4.1 Nano",
			Provider:   entities.TypeOpenAI,
			MaxTokens:  32768,
			InputCost:  0.10,
			OutputCost: 0.40,
		},
		{
			ID:         "o4-mini",
			Name:       "GPT-o4-mini",
			Provider:   entities.TypeOpenAI,
			MaxTokens:  100000,
			InputCost:  1.10,
			OutputCost: 4.40,
		},
		{
			ID:         "gpt-4o",
			Name:       "GPT-4o",
			Provider:   entities.TypeOpenAI,
			MaxTokens:  16384,
			InputCost:  2.50,
			OutputCost: 10.00,
		},
	},

	entities.TypeAnthropic: {
		{
			ID:         "claude-sonnet-4-20250514",
			Name:       "Claude Sonnet 4",
			Provider:   entities.TypeAnthropic,
			MaxTokens:  200000,
			InputCost:  3.00,
			OutputCost: 15.00,
		},
		{
			ID:         "claude-3-5-sonnet-20241022",
			Name:       "Claude 3.5 Sonnet",
			Provider:   entities.TypeAnthropic,
			MaxTokens:  200000,
			InputCost:  3.00,
			OutputCost: 15.00,
		},
		{
			ID:         "claude-3-5-haiku-20241022",
			Name:       "Claude 3.5 Haiku",
			Provider:   entities.TypeAnthropic,
			MaxTokens:  200000,
			InputCost:  0.80,
			OutputCost: 4.00,
		},
		{
			ID:         "claude-3-opus-20240229",
			Name:       "Claude 3 Opus",
			Provider:   entities.TypeAnthropic,
			MaxTokens:  200000,
			InputCost:  15.00,
			OutputCost: 75.00,
		},
		{
			ID:         "claude-3-haiku-20240307",
			Name:       "Claude 3 Haiku",
			Provider:   entities.TypeAnthropic,
			MaxTokens:  200000,
			InputCost:  0.25,
			OutputCost: 1.25,
		},
	},

	entities.TypeGoogle: {
		{
			ID:         "gemini-2.0-flash",
			Name:       "Gemini 2.0 Flash",
			Provider:   entities.TypeGoogle,
			MaxTokens:  1048576,
			InputCost:  0.10,
			OutputCost: 0.40,
		},
		{
			ID:         "gemini-1.5-pro",
			Name:       "Gemini 1.5 Pro",
			Provider:   entities.TypeGoogle,
			MaxTokens:  2097152,
			InputCost:  1.25,
			OutputCost: 5.00,
		},
		{
			ID:         "gemini-1.5-flash",
			Name:       "Gemini 1.5 Flash",
			Provider:   entities.TypeGoogle,
			MaxTokens:  1048576,
			InputCost:  0.075,
			OutputCost: 0.30,
		},
		{
			ID:         "gemini-1.5-flash-8b",
			Name:       "Gemini 1.5 Flash 8B",
			Provider:   entities.TypeGoogle,
			MaxTokens:  1048576,
			InputCost:  0.0375,
			OutputCost: 0.15,
		},
	},
}

func GetModelInfo(providerType entities.Type, modelID string) *entities.Model {
	models, exists := availableModels[providerType]
	if !exists {
		return nil
	}

	for _, m := range models {
		if m.ID == modelID {
			return m
		}
	}

	return nil
}

func CalculateCost(providerType entities.Type, modelID string, inputTokens, outputTokens int) float64 {
	model := GetModelInfo(providerType, modelID)
	if model == nil {
		return 0
	}

	inputCost := (float64(inputTokens) / 1_000_000) * model.InputCost
	outputCost := (float64(outputTokens) / 1_000_000) * model.OutputCost

	return inputCost + outputCost
}
