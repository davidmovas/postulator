package models

import "github.com/davidmovas/postulator/internal/kernel/dto"

type ListModelsRequest struct{}

type ListModelsResponse struct {
	Models []Model `json:"models"`
}

type UpsertModelRequest struct {
	Provider               string  `json:"provider" description:"The provider, such as openai"`
	Model                  string  `json:"model" description:"The model name the provider answers to"`
	ContextTokens          int     `json:"contextTokens" minimum:"1" description:"Tokens the model reads in one call"`
	MaxOutputTokens        int     `json:"maxOutputTokens" minimum:"1" description:"Tokens it may answer with"`
	InputUSDPerM           float64 `json:"inputUsdPerM,omitempty" minimum:"0" description:"USD per million input tokens"`
	CachedInputUSDPerM     float64 `json:"cachedInputUsdPerM,omitempty" minimum:"0" description:"Cache read price; left out, the input price"`
	CacheWriteUSDPerM      float64 `json:"cacheWriteUsdPerM,omitempty" minimum:"0" description:"Cache write price; left out, the input price"`
	OutputUSDPerM          float64 `json:"outputUsdPerM,omitempty" minimum:"0" description:"USD per million output tokens"`
	FlexInputUSDPerM       float64 `json:"flexInputUsdPerM,omitempty" minimum:"0" description:"Flex input price; left out, no flex"`
	FlexCachedInputUSDPerM float64 `json:"flexCachedInputUsdPerM,omitempty" minimum:"0" description:"Flex cache read price"`
	FlexCacheWriteUSDPerM  float64 `json:"flexCacheWriteUsdPerM,omitempty" minimum:"0" description:"Flex cache write price"`
	FlexOutputUSDPerM      float64 `json:"flexOutputUsdPerM,omitempty" minimum:"0" description:"Flex output price"`
	RPM                    int     `json:"rpm" minimum:"1" description:"Requests per minute the account may make"`
	TPM                    int     `json:"tpm" minimum:"1" description:"Tokens per minute the account may use"`
	SupportsStructured     bool    `json:"supportsStructured,omitempty" description:"It can answer against a JSON schema"`
	SupportsImages         bool    `json:"supportsImages,omitempty" description:"It can read images"`
	Reasoning              bool    `json:"reasoning,omitempty" description:"It bills its thinking as output"`
	ReasoningEffort        string  `json:"reasoningEffort,omitempty" enum:"none,low,medium,high,xhigh" description:"How long it may think"`
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

type SpendReportRequest struct {
	Days  int    `json:"days,omitempty" minimum:"0" maximum:"366" description:"How many days back to count, 1 to 366; leave it out for 30. A run is counted whole"`
	RunID string `json:"runId,omitempty" description:"Count only this run, step by step; leave it out for every call in the range"`
}

type SpendReportResponse struct {
	Since  dto.Time     `json:"since"`
	Days   int          `json:"days"`
	RunID  string       `json:"runId"`
	Totals SpendTotals  `json:"totals"`
	Slices []SpendSlice `json:"slices"`
}

type ListCallsRequest struct {
	dto.ListRequest
	RunID          string `json:"runId,omitempty" description:"Keep only this run's calls"`
	ConversationID string `json:"conversationId,omitempty" description:"Keep only this conversation's calls"`
}
