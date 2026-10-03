package models

import (
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/dto"
)

type ModelRef struct {
	Provider string `json:"provider"`
	Model    string `json:"model"`
}

type Prices struct {
	InputUSDPerM           float64 `json:"inputUsdPerM"`
	CachedInputUSDPerM     float64 `json:"cachedInputUsdPerM"`
	CacheWriteUSDPerM      float64 `json:"cacheWriteUsdPerM"`
	OutputUSDPerM          float64 `json:"outputUsdPerM"`
	FlexInputUSDPerM       float64 `json:"flexInputUsdPerM"`
	FlexCachedInputUSDPerM float64 `json:"flexCachedInputUsdPerM"`
	FlexCacheWriteUSDPerM  float64 `json:"flexCacheWriteUsdPerM"`
	FlexOutputUSDPerM      float64 `json:"flexOutputUsdPerM"`
}

type Model struct {
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	ContextTokens   int    `json:"contextTokens"`
	MaxOutputTokens int    `json:"maxOutputTokens"`
	Prices
	RPM                int    `json:"rpm"`
	TPM                int    `json:"tpm"`
	SupportsStructured bool   `json:"supportsStructured"`
	SupportsImages     bool   `json:"supportsImages"`
	Reasoning          bool   `json:"reasoning"`
	ReasoningEffort    string `json:"reasoningEffort"`
}

type Profile struct {
	Global    *ModelRef `json:"global,omitempty"`
	Effective *ModelRef `json:"effective,omitempty"`
	Role      string    `json:"role"`
}

type Usage struct {
	Input       int `json:"input"`
	CachedInput int `json:"cachedInput"`
	CacheWrite  int `json:"cacheWrite"`
	Output      int `json:"output"`
	Reasoning   int `json:"reasoning"`
	Total       int `json:"total"`
}

type Spent struct {
	Calls       int     `json:"calls"`
	Failed      int     `json:"failed"`
	Input       int     `json:"input"`
	CachedInput int     `json:"cachedInput"`
	CacheWrite  int     `json:"cacheWrite"`
	Output      int     `json:"output"`
	Reasoning   int     `json:"reasoning"`
	USD         float64 `json:"usd"`
}

type SpendTotals struct {
	Spent
	CachedShare    float64 `json:"cachedShare"`
	ReasoningShare float64 `json:"reasoningShare"`
	FlexShare      float64 `json:"flexShare"`
}

type SpendSlice struct {
	Purpose  string `json:"purpose"`
	Provider string `json:"provider"`
	Model    string `json:"model"`
	Tier     string `json:"tier"`
	Step     string `json:"step"`
	Spent
}

type Call struct {
	ID        string   `json:"id"`
	CreatedAt dto.Time `json:"createdAt"`
	Purpose   string   `json:"purpose"`
	Step      string   `json:"step"`
	Provider  string   `json:"provider"`
	Model     string   `json:"model"`
	Tier      string   `json:"tier"`
	Usage
	USD            float64 `json:"usd"`
	LatencyMs      int64   `json:"latencyMs"`
	Status         string  `json:"status"`
	ErrorCode      string  `json:"errorCode"`
	RunID          string  `json:"runId"`
	ItemID         string  `json:"itemId"`
	ConversationID string  `json:"conversationId"`
}

type ProviderKey struct {
	Provider   string `json:"provider"`
	Configured bool   `json:"configured"`
}

func modelView(info llm.ModelInfo) Model {
	return Model{
		Provider:        info.Ref.Provider,
		Model:           info.Ref.Model,
		ContextTokens:   info.ContextTokens,
		MaxOutputTokens: info.MaxOutputTokens,
		Prices: Prices{
			InputUSDPerM:           info.InputUSDPerM,
			CachedInputUSDPerM:     info.CachedInputUSDPerM,
			CacheWriteUSDPerM:      info.CacheWriteUSDPerM,
			OutputUSDPerM:          info.OutputUSDPerM,
			FlexInputUSDPerM:       info.FlexInputUSDPerM,
			FlexCachedInputUSDPerM: info.FlexCachedInputUSDPerM,
			FlexCacheWriteUSDPerM:  info.FlexCacheWriteUSDPerM,
			FlexOutputUSDPerM:      info.FlexOutputUSDPerM,
		},
		RPM:                info.RPM,
		TPM:                info.TPM,
		SupportsStructured: info.SupportsStructured,
		SupportsImages:     info.SupportsImages,
		Reasoning:          info.Reasoning,
		ReasoningEffort:    string(info.ReasoningEffort),
	}
}

func refView(ref llm.ModelRef) *ModelRef {
	if !ref.Valid() {
		return nil
	}
	return &ModelRef{Provider: ref.Provider, Model: ref.Model}
}

func usageView(usage llm.Usage) Usage {
	return Usage{
		Input: usage.Input, CachedInput: usage.CachedInput, CacheWrite: usage.CacheWrite, Output: usage.Output,
		Reasoning: usage.Reasoning, Total: usage.Total,
	}
}

func spentView(slice llm.SpendSlice) Spent {
	return Spent{
		Calls: slice.Calls, Failed: slice.Failed, Input: slice.Input, CachedInput: slice.CachedInput,
		CacheWrite: slice.CacheWrite, Output: slice.Output, Reasoning: slice.Reasoning, USD: slice.USD,
	}
}

func (s Spent) plus(other Spent) Spent {
	return Spent{
		Calls: s.Calls + other.Calls, Failed: s.Failed + other.Failed, Input: s.Input + other.Input,
		CachedInput: s.CachedInput + other.CachedInput, CacheWrite: s.CacheWrite + other.CacheWrite,
		Output: s.Output + other.Output, Reasoning: s.Reasoning + other.Reasoning, USD: s.USD + other.USD,
	}
}

func sliceView(slice llm.SpendSlice) SpendSlice {
	return SpendSlice{
		Purpose: string(slice.Purpose), Provider: slice.Provider, Model: slice.Model, Tier: string(slice.Tier),
		Step: slice.Step, Spent: spentView(slice),
	}
}

func totalsOf(slices []llm.SpendSlice) SpendTotals {
	var (
		totals SpendTotals
		flex   float64
	)
	for i := range slices {
		totals.Spent = totals.plus(spentView(slices[i]))
		if slices[i].Tier == llm.TierFlex {
			flex += slices[i].USD
		}
	}
	totals.CachedShare = share(float64(totals.CachedInput), float64(totals.Input))
	totals.ReasoningShare = share(float64(totals.Reasoning), float64(totals.Output))
	totals.FlexShare = share(flex, totals.USD)
	return totals
}

func share(part, whole float64) float64 {
	if whole <= 0 {
		return 0
	}
	return part / whole
}

func callView(call llm.Call) Call {
	return Call{
		ID:             call.ID,
		CreatedAt:      dto.NewTime(call.CreatedAt),
		Purpose:        string(llm.PurposeOf(call.RunID, call.Step)),
		Step:           call.Step,
		Provider:       call.Ref.Provider,
		Model:          call.Ref.Model,
		Tier:           string(call.Tier),
		Usage:          usageView(call.Usage),
		USD:            call.USD,
		LatencyMs:      call.Latency.Milliseconds(),
		Status:         string(call.Status),
		ErrorCode:      call.ErrorCode,
		RunID:          call.RunID,
		ItemID:         call.ItemID,
		ConversationID: call.ConversationID,
	}
}
