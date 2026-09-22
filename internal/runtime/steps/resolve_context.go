package steps

import (
	"context"
	"strconv"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
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

			lc := content.PlanLinks(g, pagemap.NewIndex(pages), content.Subject{
				Site:     pagemap.NewSite(owner.BaseURL),
				PageID:   sc.Page.ID,
				PagePath: sc.Page.Path,
				EntityID: entity.ID,
			}, policy).Context
			blob, err := encode(lc, "link context")
			if err != nil {
				return run.Result{}, err
			}

			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactLinkContext, Blob: blob}},
				Message:   "resolved " + plural(len(lc.Targets)) + " for " + sc.Page.Path,
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
