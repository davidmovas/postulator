package steps

import (
	"context"
	"embed"

	"github.com/davidmovas/postulator/internal/application/llm"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
)

//go:embed prompts/*.tmpl
var promptFS embed.FS

var prompts = llm.MustPrompts(promptFS, "prompts/*.tmpl")

type modelCall struct {
	ref  domainllm.ModelRef
	step string
	role domainllm.Role
}

func modelFor(ctx context.Context, deps Deps, sc *run.StepContext, step string, role domainllm.Role) (modelCall, error) {
	ref, err := deps.Profiles.Resolve(ctx, sc.Run.SiteID, role, sc.Spec.ModelProfiles)
	if err != nil {
		return modelCall{}, err
	}
	return modelCall{ref: ref, step: step, role: role}, nil
}

func stepRequest(sc *run.StepContext, call modelCall, prompt any, maxTokens int) (llm.Request, error) {
	system, user, err := prompts.Render(call.step, prompt)
	if err != nil {
		return llm.Request{}, err
	}
	return llm.Request{
		Ref:       call.ref,
		System:    system,
		Messages:  []llm.Message{{Role: llm.RoleUser, Text: user}},
		MaxTokens: maxTokens,
		Meta:      callMeta(sc, call.step, call.role),
	}, nil
}

func callMeta(sc *run.StepContext, step string, role domainllm.Role) llm.CallMeta {
	return llm.CallMeta{RunID: sc.Run.ID, ItemID: sc.Item.ID, Step: step, Role: role}
}
