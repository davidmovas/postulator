package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const runsStartName = "runs_start"

type runsStartArgs struct {
	PageIDs     []string `json:"pageIds" description:"Page ids from a read tool"`
	TemplateID  string   `json:"templateId,omitempty" description:"Template the pages move to; left out, each keeps its own"`
	Steps       []string `json:"steps,omitempty" enum:"resolve_context,generate_body,generate_meta,insert_links,repair_links,generate_images,validate,judge,publish,relink_neighbors,sync_back,report" description:"Steps in order; left out, the template's recipe"`
	PublishMode string   `json:"publishMode,omitempty" enum:"draft,publish" description:"Leave a WordPress draft (the default) or publish"`
	Kind        string   `json:"kind,omitempty" enum:"generate,relink,audit,sync,import,repair,revert,custom" description:"What the run is for; left out, the steps decide"`
	MaxUSD      float64  `json:"maxUsd,omitempty" minimum:"0" description:"Dollar ceiling; left out, the configured one"`
	MaxTokens   int      `json:"maxTokens,omitempty" minimum:"0" description:"Token ceiling; left out, the configured one"`
}

func runsStart(deps Deps) Tool {
	return newSiteTool(Def{
		Name:        runsStartName,
		Description: "Start a run over the named pages and return its run id at once.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in runsStartArgs) (runs.StartResponse, error) {
		return deps.Runs.Start(ctx, runs.StartRequest{
			SiteID:      b.SiteID,
			PageIDs:     in.PageIDs,
			TemplateID:  in.TemplateID,
			Recipe:      recipeOf(in.Steps),
			PublishMode: in.PublishMode,
			Kind:        in.Kind,
			Budget:      run.Budget{MaxUSD: in.MaxUSD, MaxTokens: in.MaxTokens},
		})
	})
}

func recipeOf(steps []string) []template.StepSpec {
	if len(steps) == 0 {
		return nil
	}

	recipe := make([]template.StepSpec, 0, len(steps))
	for _, step := range steps {
		recipe = append(recipe, template.StepSpec{Name: step, Enabled: true})
	}
	return recipe
}
