package agent

import (
	"embed"

	"github.com/davidmovas/postulator/internal/application/llm"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")
