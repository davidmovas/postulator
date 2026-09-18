package steps

import (
	"embed"

	"github.com/davidmovas/postulator/internal/application/llm"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")

func render(step string, data any) (system, user string, err error) {
	return prompts.Render(step, data)
}
