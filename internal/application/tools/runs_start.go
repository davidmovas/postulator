package tools

import (
	"context"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const runsStartName = "runs_start"

type runsStartArgs struct {
	PageIDs     []string `json:"pageIds"`
	TemplateID  string   `json:"templateId,omitempty"`
	Steps       []string `json:"steps,omitempty" description:"the step names to run in order; the recipe of the resolved template is used when this is empty"`
	PublishMode string   `json:"publishMode,omitempty" enum:"draft,publish"`
	Kind        string   `json:"kind,omitempty" enum:"generate,relink,audit,custom"`
	MaxUSD      float64  `json:"maxUsd,omitempty"`
	MaxTokens   int      `json:"maxTokens,omitempty"`
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
