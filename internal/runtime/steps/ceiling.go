package steps

import (
	"github.com/davidmovas/postulator/internal/domain/template"
)

const (
	tokensPerWord     = 3
	tokenHeadroom     = 1024
	leastWriterTokens = 4096
	fallbackWords     = 800
	ceilingDoublings  = 1
)

func plannedWords(spec template.TemplateSpec) int {
	words := 0
	for i := range spec.Sections {
		words += spec.Sections[i].TargetWords
	}
	if words == 0 {
		words = spec.Length.Max
	}
	if words == 0 {
		words = fallbackWords
	}
	return words
}

func writerCeiling(spec template.TemplateSpec, attempts int) int {
	ceiling := max(leastWriterTokens, plannedWords(spec)*tokensPerWord+tokenHeadroom)
	return ceiling << min(max(attempts, 0), ceilingDoublings)
}

func atFullRoom(attempts int) bool {
	return attempts >= ceilingDoublings
}
