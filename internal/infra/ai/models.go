package ai

import "github.com/davidmovas/postulator/internal/domain/entities"

var availableModels = map[entities.Type][]*entities.Model{
	entities.TypeOpenAI: {
		// GPT-5.2 - Best for coding and agentic tasks
		{
			ID:                   "gpt-5.2",
			Name:                 "GPT-5.2",
			Provider:             entities.TypeOpenAI,
			ContextWindow:        400000,
			MaxOutputTokens:      128000,
			InputCost:            1.75,
			OutputCost:           14.00,
			RPM:                  500,
			TPM:                  500000,
			UsesCompletionTokens: true,
			IsReasoningModel:     true,
		},
		// GPT-5 Mini - Fast and efficient
		{
			ID:                   "gpt-5-mini",
			Name:                 "GPT-5 Mini",
			Provider:             entities.TypeOpenAI,
			ContextWindow:        400000,
			MaxOutputTokens:      128000,
			InputCost:            0.25,
			OutputCost:           2.00,
			RPM:                  500,
			TPM:                  500000,
			UsesCompletionTokens: true,
			IsReasoningModel:     true,
		},
		// GPT-5 Nano - Cheapest option
		{
			ID:                   "gpt-5-nano",
			Name:                 "GPT-5 Nano",
			Provider:             entities.TypeOpenAI,
			ContextWindow:        400000,
			MaxOutputTokens:      128000,
			InputCost:            0.05,
			OutputCost:           0.40,
			RPM:                  500,
			TPM:                  200000,
			UsesCompletionTokens: true,
			IsReasoningModel:     true,
		},
		// GPT-4o Mini - Legacy fast model
		{
			ID:              "gpt-4o-mini",
			Name:            "GPT-4o Mini",
			Provider:        entities.TypeOpenAI,
			ContextWindow:   128000,
			MaxOutputTokens: 16384,
			InputCost:       0.15,
			OutputCost:      0.60,
			RPM:             500,
			TPM:             200000,
		},
		// GPT-4.1 - Large context model
		{
			ID:                   "gpt-4.1",
			Name:                 "GPT-4.1",
			Provider:             entities.TypeOpenAI,
			ContextWindow:        1000000,
			MaxOutputTokens:      32768,
			InputCost:            2.00,
			OutputCost:           8.00,
			RPM:                  500,
			TPM:                  30000,
			UsesCompletionTokens: true,
		},
		// GPT-4.1 Mini - Fast large context model
		{
			ID:                   "gpt-4.1-mini",
			Name:                 "GPT-4.1 Mini",
			Provider:             entities.TypeOpenAI,
			ContextWindow:        1000000,
			MaxOutputTokens:      32768,
			InputCost:            0.40,
			OutputCost:           1.60,
			RPM:                  500,
			TPM:                  200000,
			UsesCompletionTokens: true,
		},
	},

	entities.TypeAnthropic: {
		{
			ID:              "claude-sonnet-4-20250514",
			Name:            "Claude Sonnet 4",
			Provider:        entities.TypeAnthropic,
			ContextWindow:   200000,
			MaxOutputTokens: 8192,
			InputCost:       3.00,
			OutputCost:      15.00,
			RPM:             50,
			TPM:             40000,
		},
		{
			ID:              "claude-3-5-sonnet-20241022",
			Name:            "Claude 3.5 Sonnet",
			Provider:        entities.TypeAnthropic,
			ContextWindow:   200000,
			MaxOutputTokens: 8192,
			InputCost:       3.00,
			OutputCost:      15.00,
			RPM:             50,
			TPM:             40000,
		},
		{
			ID:              "claude-3-5-haiku-20241022",
			Name:            "Claude 3.5 Haiku",
			Provider:        entities.TypeAnthropic,
			ContextWindow:   200000,
			MaxOutputTokens: 8192,
			InputCost:       0.80,
			OutputCost:      4.00,
			RPM:             50,
			TPM:             50000,
		},
	},

	entities.TypeGoogle: {
		{
			ID:              "gemini-2.0-flash",
			Name:            "Gemini 2.0 Flash",
			Provider:        entities.TypeGoogle,
			ContextWindow:   1048576,
			MaxOutputTokens: 8192,
			InputCost:       0.10,
			OutputCost:      0.40,
			RPM:             60,
			TPM:             100000,
		},
		{
			ID:              "gemini-1.5-pro",
			Name:            "Gemini 1.5 Pro",
			Provider:        entities.TypeGoogle,
			ContextWindow:   2097152,
			MaxOutputTokens: 8192,
			InputCost:       1.25,
			OutputCost:      5.00,
			RPM:             60,
			TPM:             100000,
		},
		{
			ID:              "gemini-1.5-flash",
			Name:            "Gemini 1.5 Flash",
			Provider:        entities.TypeGoogle,
			ContextWindow:   1048576,
			MaxOutputTokens: 8192,
			InputCost:       0.075,
			OutputCost:      0.30,
			RPM:             60,
			TPM:             100000,
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
