package models

type ListModelsRequest struct{}

type ListModelsResponse struct {
	Models []Model `json:"models"`
}

type UpsertModelRequest struct {
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

type UpsertModelResponse struct {
	Model Model `json:"model"`
}

type DisableModelRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type DisableModelResponse struct{}

type SetProfileRequest struct {
	Role     string `json:"role"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type SetProfileResponse struct {
	Profile Profile `json:"profile"`
}

type GetProfilesRequest struct {
	SiteID string `json:"siteId,omitempty"`
}

type GetProfilesResponse struct {
	Profiles []Profile `json:"profiles"`
}

type SetProviderKeyRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"apiKey"`
}

type SetProviderKeyResponse struct {
	Provider string `json:"provider"`
}

type ProviderKeysRequest struct{}

type ProviderKeysResponse struct {
	Providers []ProviderKey `json:"providers"`
}

type DeleteProviderKeyRequest struct {
	Provider string `json:"provider"`
}

type DeleteProviderKeyResponse struct {
	Provider string `json:"provider"`
}

type TestProviderRequest struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type TestProviderResponse struct {
	Model     ModelRef `json:"model"`
	Usage     Usage    `json:"usage"`
	LatencyMs int64    `json:"latencyMs"`
}

type UsageSummaryRequest struct {
	RunID          string `json:"runId,omitempty"`
	ConversationID string `json:"conversationId,omitempty"`
}

type UsageSummaryResponse struct {
	Usage Usage   `json:"usage"`
	USD   float64 `json:"usd"`
	Calls int     `json:"calls"`
}
