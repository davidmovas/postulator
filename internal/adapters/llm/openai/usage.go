package openai

import (
	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/llm"
)

const reasonContentFilter = "content_filter"

type wireUsage struct {
	InputTokens        int `json:"input_tokens"`
	InputTokensDetails struct {
		CachedTokens     int `json:"cached_tokens"`
		CacheWriteTokens int `json:"cache_write_tokens"`
	} `json:"input_tokens_details"`
	OutputTokens        int `json:"output_tokens"`
	OutputTokensDetails struct {
		ReasoningTokens int `json:"reasoning_tokens"`
	} `json:"output_tokens_details"`
}

func usageOf(usage *wireUsage) llm.Usage {
	if usage == nil {
		return llm.Usage{}
	}
	return llm.Usage{
		Input:       usage.InputTokens,
		CachedInput: usage.InputTokensDetails.CachedTokens,
		CacheWrite:  usage.InputTokensDetails.CacheWriteTokens,
		Output:      usage.OutputTokens,
		Reasoning:   usage.OutputTokensDetails.ReasoningTokens,
		Total:       usage.InputTokens + usage.OutputTokens,
	}
}

func finishOf(decoded wireResponse) port.FinishReason {
	if refused(decoded.Output) {
		return port.FinishContentFilter
	}
	if decoded.Status != statusIncomplete {
		return port.FinishStop
	}
	if decoded.IncompleteDetails != nil && decoded.IncompleteDetails.Reason == reasonContentFilter {
		return port.FinishContentFilter
	}
	return port.FinishLength
}

func tierOf(served, sent string) llm.ServiceTier {
	switch served {
	case tierFlex:
		return llm.TierFlex
	case "":
		if sent == tierFlex {
			return llm.TierFlex
		}
		return llm.TierDefault
	default:
		return llm.TierDefault
	}
}
