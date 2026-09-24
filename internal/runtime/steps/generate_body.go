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
	Page     pagemap.Page
	Entity   graph.Entity
	Spec     template.TemplateSpec
	Brief    content.Brief
	Children []string
}

func childAnchors(lc content.LinkContext, wanted bool) []string {
	if !wanted {
		return nil
	}

	out := make([]string, 0, len(lc.Targets))
	for i := range lc.Targets {
		if lc.Targets[i].Relation != content.RelationDown || len(lc.Targets[i].Anchors) == 0 {
			continue
		}
		out = append(out, lc.Targets[i].Anchors[0])
	}
	return out
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
			policy, err := effectivePolicy(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			ref, err := deps.Profiles.Resolve(ctx, sc.Run.SiteID, domainllm.RoleWriter, sc.Spec.ModelProfiles)
			if err != nil {
				return run.Result{}, err
			}

			brief := content.NewBrief(sc.Spec, policy.Rules, sc.Page, entity, lc)
			system, user, err := render(NameGenerateBody, bodyPrompt{
				Page: sc.Page, Entity: entity, Spec: sc.Spec, Brief: brief,
				Children: childAnchors(lc, sc.Spec.LinkRules.ChildrenSection),
			})
			if err != nil {
				return run.Result{}, err
			}

			answer, usage, err := port.Structured[content.DraftAnswer](ctx, deps.LLM, port.Request{
				Ref:       ref,
				System:    system,
				Messages:  []port.Message{{Role: port.RoleUser, Text: user}},
				MaxTokens: writerCeiling(sc.Spec, sc.Item.Attempts),
				Meta:      callMeta(sc, NameGenerateBody),
			})
			if err != nil {
				return run.Result{}, err
			}

			draft, doc, err := content.Assemble(answer, brief)
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
