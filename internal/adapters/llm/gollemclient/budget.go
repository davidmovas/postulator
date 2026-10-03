package gollemclient

import "github.com/davidmovas/postulator/internal/domain/llm"

func budget(asked int, info llm.ModelInfo) int {
	if asked <= 0 {
		return asked
	}

	ceiling := asked
	if info.Reasoning {
		ceiling += llm.Allowance(info.ReasoningEffort)
	}
	if info.MaxOutputTokens > 0 && ceiling > info.MaxOutputTokens {
		return info.MaxOutputTokens
	}
	return ceiling
}
