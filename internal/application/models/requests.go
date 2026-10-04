package models

import "github.com/davidmovas/postulator/internal/kernel/dto"

type ListModelsRequest struct{}

type ListModelsResponse struct {
	Models []Model `json:"models"`
}

type UpsertModelRequest struct {
	Provider               string  `json:"provider" description:"Always openai"`
	Model                  string  `json:"model" description:"Provider's model name"`
	ContextTokens          int     `json:"contextTokens" minimum:"1" description:"Context window in tokens"`
	MaxOutputTokens        int     `json:"maxOutputTokens" minimum:"1" description:"Max output tokens"`
	InputUSDPerM           float64 `json:"inputUsdPerM,omitempty" minimum:"0" description:"USD per 1M input tokens"`
	CachedInputUSDPerM     float64 `json:"cachedInputUsdPerM,omitempty" minimum:"0" description:"Cache read USD/1M; default input"`
	CacheWriteUSDPerM      float64 `json:"cacheWriteUsdPerM,omitempty" minimum:"0" description:"Cache write USD/1M; default input"`
	OutputUSDPerM          float64 `json:"outputUsdPerM,omitempty" minimum:"0" description:"USD per 1M output tokens"`
	FlexInputUSDPerM       float64 `json:"flexInputUsdPerM,omitempty" minimum:"0" description:"Flex input USD/1M; omit for no flex"`
	FlexCachedInputUSDPerM float64 `json:"flexCachedInputUsdPerM,omitempty" minimum:"0" description:"Flex cache read USD/1M"`
	FlexCacheWriteUSDPerM  float64 `json:"flexCacheWriteUsdPerM,omitempty" minimum:"0" description:"Flex cache write USD/1M"`
	FlexOutputUSDPerM      float64 `json:"flexOutputUsdPerM,omitempty" minimum:"0" description:"Flex output USD/1M"`
	RPM                    int     `json:"rpm" minimum:"1" description:"Requests per minute allowed"`
	TPM                    int     `json:"tpm" minimum:"1" description:"Tokens per minute allowed"`
	SupportsStructured     bool    `json:"supportsStructured,omitempty" description:"Answers to a JSON schema"`
	SupportsImages         bool    `json:"supportsImages,omitempty" description:"Reads images"`
	Reasoning              bool    `json:"reasoning,omitempty" description:"Bills thinking as output"`
}

type UpsertModelResponse struct {
	Model Model `json:"model"`
}

type DisableModelRequest struct {
	Provider string `json:"provider" description:"Provider, openai"`
	Model    string `json:"model" description:"Model to disable"`
}

type DisableModelResponse struct{}

type SetProfileRequest struct {
	Role     string `json:"role" enum:"writer,editor,linker,judge,chat,image,titler" description:"writer drafts, editor rewrites, linker links, judge grades, chat answers here, image draws, titler names chats"`
	Provider string `json:"provider" description:"Provider, openai"`
	Model    string `json:"model" description:"Model name"`
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
	Provider string `json:"provider" description:"Provider, openai"`
	APIKey   string `json:"apiKey" description:"The key; sealed, never read back"`
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
	Provider string `json:"provider" description:"Provider to probe, openai"`
	Model    string `json:"model,omitempty" description:"Model to probe; default the cheapest enabled"`
}

type TestProviderResponse struct {
	Model     ModelRef `json:"model"`
	Usage     Usage    `json:"usage"`
	LatencyMs int64    `json:"latencyMs"`
}

type UsageSummaryRequest struct {
	RunID          string `json:"runId,omitempty" description:"Only this run's spend"`
	ConversationID string `json:"conversationId,omitempty" description:"Only this conversation's spend"`
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
