package steps

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const NameResolveContext = string(run.StepResolveContext)

func ResolveContext(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameResolveContext,
		Produces: []run.ArtifactKind{run.ArtifactLinkContext},
		Retry:    run.RetryPolicy{Max: 3},
		Timeout:  pureStepTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			entities, err := deps.Entities.ListBySite(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			edges, err := deps.Edges.ListBySite(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}
			policy, err := effectivePolicy(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			owner, err := deps.Sites.Get(ctx, sc.Run.SiteID)
			if err != nil {
				return run.Result{}, err
			}

			g, err := graph.New(entities, edges)
			if err != nil {
				return run.Result{}, err
			}

			if _, held := g.Entity(entity.ID); !held {
				return run.Result{}, errors.New(errors.Invalid,
					"the graph does not hold the entity "+sc.Page.Path+" is mapped to, so nothing says what it must link to").
					WithDetail("pageId", sc.Page.ID).
					WithDetail("path", sc.Page.Path).
					WithDetail("entityId", entity.ID)
			}

			plan := content.PlanLinks(g, pagemap.NewIndex(pages), content.Subject{
				Site:     pagemap.NewSite(owner.BaseURL),
				PageID:   sc.Page.ID,
				PagePath: sc.Page.Path,
				EntityID: entity.ID,
			}, policy)
			blob, err := encode(plan.Context, "link context")
			if err != nil {
				return run.Result{}, err
			}

			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactLinkContext, Blob: blob}},
				Message: "resolved " + plural(len(plan.Context.Targets)) + " for " + sc.Page.Path +
					withheld(plan.Blocked),
			}, nil
		},
	}
}

func plural(count int) string {
	if count == 1 {
		return "one link target"
	}
	return strconv.Itoa(count) + " link targets"
}

func withheld(blocked []content.BlockedTarget) string {
	required := 0
	for i := range blocked {
		if blocked[i].Required {
			required++
		}
	}
	if len(blocked) == 0 {
		return ""
	}
	return ", holding back " + strconv.Itoa(len(blocked)) + " the graph asks for (" +
		strconv.Itoa(required) + " of them required) because no page carries them yet"
}
