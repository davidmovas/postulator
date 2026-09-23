package gollemclient

import "github.com/davidmovas/postulator/internal/domain/llm"

const (
	allowanceLow    = 2048
	allowanceMedium = 8192
	allowanceHigh   = 24576
	allowanceXHigh  = 49152
)

func allowance(effort llm.ReasoningEffort) int {
	switch effort {
	case llm.EffortNone:
		return 0
	case llm.EffortLow:
		return allowanceLow
	case llm.EffortHigh:
		return allowanceHigh
	case llm.EffortXHigh:
		return allowanceXHigh
	case llm.EffortMedium:
		return allowanceMedium
	default:
		return allowanceMedium
	}
}

func budget(asked int, info llm.ModelInfo) int {
	if asked <= 0 || !info.Reasoning {
		return asked
	}

	ceiling := asked + allowance(info.ReasoningEffort)
	if info.MaxOutputTokens > 0 && ceiling > info.MaxOutputTokens {
		return info.MaxOutputTokens
	}
	return ceiling
}
