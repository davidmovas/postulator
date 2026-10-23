package agent

import (
	"embed"

	"github.com/davidmovas/postulator/internal/application/llm"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")

const (
	instructionsPrompt = "chat.instructions"
	situationPrompt    = "chat.context"
)

func Instructions(situation SiteContext) (string, error) {
	return prompts.One(instructionsPrompt, situation)
}

func Situation(situation SiteContext) (string, error) {
	return prompts.One(situationPrompt, situation)
}
