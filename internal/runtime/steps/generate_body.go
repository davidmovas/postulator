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

const NameGenerateBody = string(run.StepGenerateBody)

type bodyPrompt struct {
	Page    pagemap.Page
	Entity  graph.Entity
	Spec    template.TemplateSpec
	Brief   content.Brief
	Product bool
}

func GenerateBody(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameGenerateBody,
		Role:     domainllm.RoleWriter,
		Requires: []run.ArtifactKind{run.ArtifactLinkContext},
		Produces: []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
		Retry:    run.RetryPolicy{Max: 3},
		Timeout:  writerTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			lc, err := linkContextOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			policy, err := effectivePolicy(ctx, deps, sc.Run.SiteID, sc.Spec)
			if err != nil {
				return run.Result{}, err
			}

			ref, err := deps.Profiles.Resolve(ctx, sc.Run.SiteID, domainllm.RoleWriter, sc.Spec.ModelProfiles)
			if err != nil {
				return run.Result{}, err
			}

			brief := content.NewBrief(sc.Spec, policy.Rules, sc.Page, entity, lc)
			request, err := stepRequest(sc, NameGenerateBody, ref, bodyPrompt{
				Page: sc.Page, Entity: entity, Spec: sc.Spec, Brief: brief, Product: sc.Page.WPType == pagemap.WPProduct,
			}, writerCeiling(sc.Spec, sc.Page.WPType, sc.Item.Attempts))
			if err != nil {
				return run.Result{}, err
			}

			draft, doc, usage, err := write(ctx, deps, request, brief)
			if err != nil {
				return run.Result{}, err
			}
			encoded, err := encode(draft, "content draft")
			if err != nil {
				return run.Result{}, err
			}
			body, err := doc.Render()
			if err != nil {
				return run.Result{}, err
			}

			return run.Result{
				Artifacts: []run.Artifact{
					{Kind: run.ArtifactDraft, Blob: encoded},
					{Kind: run.ArtifactBodyHTML, Blob: []byte(body)},
				},
				Tokens:  usage.Total,
				Message: "wrote the body of " + sc.Page.Path,
			}, nil
		},
	}
}

func write(ctx context.Context, deps Deps, request port.Request,
	brief content.Brief) (content.ContentDraft, *content.Document, domainllm.Usage, error) {
	if brief.Product == nil {
		answer, usage, err := port.Structured[content.DraftAnswer](ctx, deps.LLM, request)
		if err != nil {
			return content.ContentDraft{}, nil, usage, err
		}
		draft, doc, err := content.Assemble(answer, brief)
		return draft, doc, usage, err
	}

	answer, usage, err := port.Structured[content.ProductAnswer](ctx, deps.LLM, request)
	if err != nil {
		return content.ContentDraft{}, nil, usage, err
	}
	draft, doc, err := content.AssembleProduct(answer, brief)
	return draft, doc, usage, err
}
