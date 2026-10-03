package steps

import (
	"embed"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")

func render(step string, data any) (system, user string, err error) {
	return prompts.Render(step, data)
}

func callMeta(sc *run.StepContext, step string) llm.CallMeta {
	return llm.CallMeta{RunID: sc.Run.ID, ItemID: sc.Item.ID, Step: step}
}
