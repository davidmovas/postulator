package models

type ListModelsRequest struct{}

type ListModelsResponse struct {
	Models []Model `json:"models"`
}

type UpsertModelRequest struct {
	Provider           string  `json:"provider" description:"The provider the model belongs to, such as openai, anthropic or gemini"`
	Model              string  `json:"model" description:"The model name the provider answers to"`
	ContextTokens      int     `json:"contextTokens" minimum:"1" description:"How many tokens the model reads in one call"`
	MaxOutputTokens    int     `json:"maxOutputTokens" minimum:"1" description:"How many tokens the model may answer with, which must fit the context"`
	InputUSDPerM       float64 `json:"inputUsdPerM,omitempty" minimum:"0" description:"Dollars per million input tokens; leave it out for a model that costs nothing to read"`
	CachedInputUSDPerM float64 `json:"cachedInputUsdPerM,omitempty" minimum:"0" description:"Dollars per million input tokens served from the provider's prompt cache; leave it out and a cache read is charged at the full rate"`
	OutputUSDPerM      float64 `json:"outputUsdPerM,omitempty" minimum:"0" description:"Dollars per million output tokens; leave it out for a model that costs nothing to answer"`
	RPM                int     `json:"rpm" minimum:"1" description:"The requests per minute the account may make"`
	TPM                int     `json:"tpm" minimum:"1" description:"The tokens per minute the account may use"`
	SupportsStructured bool    `json:"supportsStructured,omitempty" description:"The model can answer against a JSON schema; leave it out for a model that cannot"`
	SupportsImages     bool    `json:"supportsImages,omitempty" description:"The model can read images; leave it out for a model that cannot"`
	Reasoning          bool    `json:"reasoning,omitempty" description:"The model thinks before it answers and bills those tokens as output; leave it out for a model that does not"`
	ReasoningEffort    string  `json:"reasoningEffort,omitempty" enum:"none,low,medium,high,xhigh" description:"How long the model may think; leave it out to send no effort at all"`
}

type UpsertModelResponse struct {
	Model Model `json:"model"`
}

type DisableModelRequest struct {
	Provider string `json:"provider" description:"The provider of the model to switch off, such as openai"`
	Model    string `json:"model" description:"The model name to switch off"`
}

type DisableModelResponse struct{}

type SetProfileRequest struct {
	Role     string `json:"role" enum:"writer,editor,linker,judge,chat,image,titler" description:"Which job the model takes: writer drafts a page, editor rewrites, linker places links, judge grades, chat answers here, image draws, titler names a conversation"`
	Provider string `json:"provider" description:"The provider to use for that job, such as openai"`
	Model    string `json:"model" description:"The model to use for that job"`
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
	Provider string `json:"provider" description:"The provider the key belongs to, such as openai, anthropic or gemini"`
	APIKey   string `json:"apiKey" description:"The key itself, which is sealed on the machine and never read back"`
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
	Provider string `json:"provider" description:"The provider to probe with one small call, such as openai"`
	Model    string `json:"model,omitempty" description:"The model to probe with, left out to take the cheapest enabled one"`
}

type TestProviderResponse struct {
	Model     ModelRef `json:"model"`
	Usage     Usage    `json:"usage"`
	LatencyMs int64    `json:"latencyMs"`
}

type UsageSummaryRequest struct {
	RunID          string `json:"runId,omitempty" description:"Count only what this run spent; leave it out for everything"`
	ConversationID string `json:"conversationId,omitempty" description:"Count only what this conversation spent; leave it out for everything"`
}

type UsageSummaryResponse struct {
	Usage Usage   `json:"usage"`
	USD   float64 `json:"usd"`
	Calls int     `json:"calls"`
}
