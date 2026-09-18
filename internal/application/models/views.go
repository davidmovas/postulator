package models

import "github.com/davidmovas/postulator/internal/domain/llm"

type ModelRef struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type Model struct {
	Provider           string  `json:"provider"`
	Model              string  `json:"model"`
	ContextTokens      int     `json:"contextTokens"`
	MaxOutputTokens    int     `json:"maxOutputTokens"`
	InputUSDPerM       float64 `json:"inputUsdPerM"`
	OutputUSDPerM      float64 `json:"outputUsdPerM"`
	RPM                int     `json:"rpm"`
	TPM                int     `json:"tpm"`
	SupportsStructured bool    `json:"supportsStructured"`
	SupportsImages     bool    `json:"supportsImages"`
	Reasoning          bool    `json:"reasoning"`
}

type Profile struct {
	Global    *ModelRef `json:"global,omitempty"`
	Effective *ModelRef `json:"effective,omitempty"`
	Role      string    `json:"role"`
}

type Usage struct {
	Input  int `json:"input"`
	Output int `json:"output"`
	Total  int `json:"total"`
}

func modelView(info llm.ModelInfo) Model {
	return Model{
		Provider:           info.Ref.Provider,
		Model:              info.Ref.Model,
		ContextTokens:      info.ContextTokens,
		MaxOutputTokens:    info.MaxOutputTokens,
		InputUSDPerM:       info.InputUSDPerM,
		OutputUSDPerM:      info.OutputUSDPerM,
		RPM:                info.RPM,
		TPM:                info.TPM,
		SupportsStructured: info.SupportsStructured,
		SupportsImages:     info.SupportsImages,
		Reasoning:          info.Reasoning,
	}
}

func refView(ref llm.ModelRef) *ModelRef {
	if !ref.Valid() {
		return nil
	}
	return &ModelRef{Provider: ref.Provider, Model: ref.Model}
}

func usageView(usage llm.Usage) Usage {
	return Usage{Input: usage.Input, Output: usage.Output, Total: usage.Total}
}
