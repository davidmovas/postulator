package content

import (
	"embed"

	"github.com/davidmovas/postulator/internal/application/llm"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")

func render(name string, data any) (system, user string, err error) {
	return prompts.Render(name, data)
}
