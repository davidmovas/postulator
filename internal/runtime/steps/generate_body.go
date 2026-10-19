package steps

import (
	"context"
	stderrors "errors"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameGenerateBody = string(run.StepGenerateBody)

	ReasonWriterOutOfRoom = "the model ran out of room before it finished, even with twice the room it is first given; " +
		"lower the word counts of the template or choose a model that writes longer answers, then retry the step"
)

type bodyPrompt struct {
	Page    pagemap.Page
	Type    string
	Entity  graph.Entity
	Spec    template.TemplateSpec
	Brief   content.Brief
	Product bool
	Picture string
}

func bodyField(sc *run.StepContext) field.Key {
	return pageKey(sc.Page)
}

func bodySchema(product bool) string {
	if product {
		return schemaText[content.ProductAnswer]()
	}
	return schemaText[content.DraftAnswer]()
}

func GenerateBody(deps Deps) run.StepDef {
	return run.StepDef{
		Name:      NameGenerateBody,
		Role:      domainllm.RoleWriter,
		Requires:  []run.ArtifactKind{run.ArtifactLinkContext},
		Produces:  []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
		Retry:     run.RetryPolicy{Max: 3},
		Timeout:   writerTimeout,
		Price:     run.Price{Input: packInput(deps, domainllm.RoleWriter, pageKey)},
		Preflight: inputPreflight(deps, domainllm.RoleWriter, pageKey),
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			lc, err := linkContextOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			call, err := modelFor(ctx, deps, sc, NameGenerateBody, domainllm.RoleWriter)
			if err != nil {
				return run.Result{}, err
			}

			brief := content.NewBrief(sc.Spec, sc.Links.Rules, sc.Page, entity, lc)
			brief.Links = content.NewLinkAllowance(lc, sc.Links)
			prompt := bodyPrompt{
				Page: sc.Page, Type: sc.Type.Name, Entity: entity, Spec: sc.Spec, Brief: brief, Product: sc.Page.WPType == pagemap.WPProduct,
			}
			ceiling := writerCeiling(sc.Spec, sc.Page.WPType, sc.Item.Attempts)
			seen, err := deps.fitted(ctx, sc, call, bodyField(sc), &lc, ceiling, func() (string, string, error) {
				return prompts.Render(NameGenerateBody, prompt)
			}, bodySchema(prompt.Product))
			if err != nil {
				return run.Result{}, err
			}
			prompt.Picture = picture(seen.Fitted, contextpack.SectionKeywords)
			request, err := stepRequest(sc, call, prompt, ceiling)
			if err != nil {
				return run.Result{}, err
			}

			draft, doc, usage, err := write(ctx, deps, request, brief)
			if truncated(err) && atFullRoom(sc.Item.Attempts) {
				return outOfRoom(sc, usage), nil
			}
			if err != nil {
				return run.Result{}, err
			}
			draft.Kept = keptBody(sc, seen)
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

func truncated(err error) bool {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) || kernel == nil || kernel.Code != errors.External {
		return false
	}
	reason, ok := kernel.Details["reason"].(string)
	return ok && reason == port.ReasonOutputTruncated
}

func outOfRoom(sc *run.StepContext, usage domainllm.Usage) run.Result {
	paused := needsHuman("the body of " + sc.Page.Path + " is unfinished: " + ReasonWriterOutOfRoom)
	paused.Tokens = usage.Total
	return paused
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
