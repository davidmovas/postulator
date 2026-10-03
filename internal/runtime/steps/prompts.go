package steps

import (
	"embed"

	"github.com/davidmovas/postulator/internal/application/llm"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")

func stepRequest(sc *run.StepContext, step string, ref domainllm.ModelRef, prompt any, maxTokens int) (llm.Request, error) {
	system, user, err := prompts.Render(step, prompt)
	if err != nil {
		return llm.Request{}, err
	}
	return llm.Request{
		Ref:       ref,
		System:    system,
		Messages:  []llm.Message{{Role: llm.RoleUser, Text: user}},
		MaxTokens: maxTokens,
		Meta:      callMeta(sc, step),
	}, nil
}

func callMeta(sc *run.StepContext, step string) llm.CallMeta {
	return llm.CallMeta{RunID: sc.Run.ID, ItemID: sc.Item.ID, Step: step}
}
