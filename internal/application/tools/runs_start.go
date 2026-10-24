package tools

import (
	"context"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const runsStartName = "runs_start"

type runsStartArgs struct {
	PageIDs     []string `json:"pageIds" description:"Page ids"`
	TemplateID  string   `json:"templateId,omitempty" description:"Template id to move pages to; default their own"`
	Steps       []string `json:"steps,omitempty" enum:"resolve_context,generate_body,generate_meta,generate_fields,insert_links,repair_links,generate_images,validate,judge,publish,relink_neighbors,sync_back,report" description:"Steps in order; default the template's recipe"`
	Fields      []string `json:"fields,omitempty" description:"Field keys the run writes, its other steps skipped; default every field"`
	Rewrite     []string `json:"rewrite,omitempty" description:"Page ids whose owner's text the run may rewrite"`
	PublishMode string   `json:"publishMode,omitempty" enum:"draft,publish" description:"Default draft"`
	Kind        string   `json:"kind,omitempty" enum:"generate,relink,audit,sync,import,custom" description:"Run purpose; default from the steps"`
	MaxUSD      float64  `json:"maxUsd,omitempty" minimum:"0" description:"Dollar ceiling; default configured"`
	MaxTokens   int      `json:"maxTokens,omitempty" minimum:"0" description:"Token ceiling; default configured"`
}

func (in *runsStartArgs) request(siteID string) runs.StartRequest {
	return runs.StartRequest{
		SiteID:      siteID,
		PageIDs:     in.PageIDs,
		TemplateID:  in.TemplateID,
		Recipe:      recipeOf(in.Steps),
		PublishMode: in.PublishMode,
		Kind:        in.Kind,
		Budget:      run.Budget{MaxUSD: in.MaxUSD, MaxTokens: in.MaxTokens},
		Rewrite:     in.Rewrite,
		Fields:      in.Fields,
	}
}

func runsStart(deps Deps) Tool {
	return confirming(newSiteTool(Def{
		Name:        runsStartName,
		Description: "Start a run over the named pages and return its run id at once.",
		Risk:        RiskWrite,
	}, func(ctx context.Context, b Binding, in runsStartArgs) (runs.StartResponse, error) {
		return deps.Runs.Start(ctx, in.request(b.SiteID))
	}), func(ctx context.Context, b Binding, in runsStartArgs) (string, error) {
		if b.Mode != agent.ModeConfirm {
			return "", nil
		}
		priced, err := deps.Runs.Estimate(ctx, in.request(b.SiteID))
		if err != nil {
			return "", err
		}
		return deps.startSummary(ctx, in, priced)
	})
}

func (deps Deps) startSummary(ctx context.Context, in runsStartArgs, priced runs.EstimateResponse) (string, error) {
	named := make([]string, 0, min(len(in.PageIDs), maxPagesNamed))
	for _, pageID := range in.PageIDs[:min(len(in.PageIDs), maxPagesNamed)] {
		path, err := deps.pathOf(ctx, pageID)
		if err != nil {
			return "", err
		}
		named = append(named, path)
	}
	pages := strings.Join(named, ", ")
	if more := len(in.PageIDs) - len(named); more > 0 {
		pages += " and " + strconv.Itoa(more) + " more"
	}
	said := "Start a " + kindOf(in) + " run over " + plural(len(in.PageIDs), "page") + " (" + pages + ")"
	if len(in.Fields) > 0 {
		said += " writing only " + strings.Join(in.Fields, ", ")
	}
	if len(in.Rewrite) > 0 {
		said += ", rewriting the owner's own text on " + plural(len(in.Rewrite), "page")
	}
	if len(priced.Added) > 0 {
		said += ", creating " + plural(len(priced.Added), "planned parent") + " first"
	}
	mode := "as drafts that wait for a review"
	if in.PublishMode == string(run.PublishLive) {
		mode = "straight to the site"
	}
	return said + ", " + mode + ": estimated $" + strconv.FormatFloat(priced.Estimate.USD, 'f', 2, 64) + " for " +
		strconv.Itoa(priced.Estimate.Tokens) + " tokens", nil
}

const maxPagesNamed = 3

func kindOf(in runsStartArgs) string {
	if in.Kind != "" {
		return in.Kind
	}
	return string(run.KindGenerate)
}

func plural(count int, noun string) string {
	if count == 1 {
		return "1 " + noun
	}
	return strconv.Itoa(count) + " " + noun + "s"
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
