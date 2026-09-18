package steps

import (
	"context"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const NameGenerateBody = "generate_body"

type bodyPrompt struct {
	Page    pagemap.Page
	Entity  graph.Entity
	Spec    template.TemplateSpec
	Phrases []string
}

func GenerateBody(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameGenerateBody,
		Role:     domainllm.RoleWriter,
		Requires: []run.ArtifactKind{run.ArtifactLinkContext},
		Produces: []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
		Retry:    run.RetryPolicy{Max: 3},
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			lc, err := linkContextOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			ref, err := deps.Profiles.Resolve(ctx, sc.Run.SiteID, domainllm.RoleWriter, sc.Spec.ModelProfiles)
			if err != nil {
				return run.Result{}, err
			}

			system, user, err := render(NameGenerateBody, bodyPrompt{
				Page: sc.Page, Entity: entity, Spec: sc.Spec, Phrases: lc.Phrases(),
			})
			if err != nil {
				return run.Result{}, err
			}

			draft, usage, err := port.Structured[content.ContentDraft](ctx, deps.LLM, port.Request{
				Ref:       ref,
				System:    system,
				Messages:  []port.Message{{Role: port.RoleUser, Text: user}},
				MaxTokens: maxTokens(sc.Spec),
				Meta:      callMeta(sc, NameGenerateBody),
			})
			if err != nil {
				return run.Result{}, err
			}

			doc, err := content.Assemble(draft)
			if err != nil {
				return run.Result{}, err
			}
			encoded, err := encode(draft, "content draft")
			if err != nil {
				return run.Result{}, err
			}

			return run.Result{
				Artifacts: []run.Artifact{
					{Kind: run.ArtifactDraft, Blob: encoded},
					{Kind: run.ArtifactBodyHTML, Blob: []byte(doc.HTML())},
				},
				Tokens:  usage.Total,
				Message: "wrote the body of " + sc.Page.Path,
			}, nil
		},
	}
}
