package steps

import (
	"context"
	"slices"
	"strconv"
	"strings"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const (
	NameValidate     = string(run.StepValidate)
	ParamAllowErrors = "allowErrors"
)

type ValidationReport struct {
	PageID     string               `json:"pageId"`
	Compliance content.Report       `json:"compliance"`
	Structure  content.Report       `json:"structure"`
	Links      content.InsertResult `json:"links"`
	Score      float64              `json:"score"`
}

func Validate(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameValidate,
		Requires: []run.ArtifactKind{run.ArtifactLinkContext, run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactValidationReport},
		Retry:    run.RetryPolicy{Max: 2},
		Timeout:  pureStepTimeout,
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			lc, err := linkContextOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			doc, err := bodyOf(sc)
			if err != nil {
				return run.Result{}, err
			}
			policy, err := effectivePolicy(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}
			entity, err := entityOf(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			links, _, err := run.Get[content.InsertResult](sc.Check, checkpointLinks)
			if err != nil {
				return run.Result{}, err
			}
			repairs, _, err := run.Get[[]content.Finding](sc.Check, CheckpointRepairs)
			if err != nil {
				return run.Result{}, err
			}
			draft, drafted, err := decodeArtifact[content.ContentDraft](sc, run.ArtifactDraft)
			if err != nil {
				return run.Result{}, err
			}

			live, err := livePages(ctx, deps, sc)
			if err != nil {
				return run.Result{}, err
			}

			report := ValidationReport{
				PageID:     sc.Page.ID,
				Compliance: content.Compliance(doc, lc, policy, sc.Page.ID),
				Structure:  content.Structure(doc, entity.PrimaryKeyword, entity.SecondaryKeywords, sc.Spec),
				Links:      links,
			}
			report.Compliance.Items = append(report.Compliance.Items, content.Unpublished(doc, lc, live, sc.Page.ID)...)
			report.Compliance.Score = content.ScoreOf(report.Compliance.Items)
			if drafted {
				report.Structure.Items = append(report.Structure.Items, draft.Findings...)
			}
			report.Structure.Items = plannedH1Wins(append(report.Structure.Items, repairs...))
			report.Structure.Score = content.ScoreOf(report.Structure.Items)
			report.Score = min(report.Compliance.Score, report.Structure.Score)

			blob, err := encode(report, "validation report")
			if err != nil {
				return run.Result{}, err
			}

			result := run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactValidationReport, Blob: blob}},
				Message:   "validation scored " + strconv.FormatFloat(report.Score, 'f', 2, 64),
			}
			clean := !report.Compliance.HasErrors() && !report.Structure.HasErrors()
			if clean || sc.BoolParam(ParamAllowErrors) || sc.Accepted() {
				return result, nil
			}

			result.Next = run.TransitionPause
			result.Reason = run.PauseNeedsHuman
			result.Message = heldMessage(report)
			return result, nil
		},
	}
}

func livePages(ctx context.Context, deps Deps, sc *run.StepContext) (map[string]bool, error) {
	pages, err := deps.Pages.ListBySite(ctx, sc.Run.SiteID)
	if err != nil {
		return nil, err
	}
	live := make(map[string]bool, len(pages)+len(sc.Run.Targets))
	for i := range pages {
		if pages[i].WPID != nil {
			live[pages[i].ID] = true
		}
	}
	for _, target := range sc.Run.Targets {
		live[target] = true
	}
	return live, nil
}

func plannedH1Wins(items []content.Finding) []content.Finding {
	planned := slices.ContainsFunc(items, func(item content.Finding) bool {
		return item.Code == content.CodePlanH1LacksKeyword
	})
	if !planned {
		return items
	}
	for i := range items {
		if items[i].Code == content.CodePrimaryMissingInH1 {
			items[i].Severity = content.SeverityWarn
			items[i].Message += ", because the planned h1 of the page does not carry it; edit the page or leave the rule off"
		}
	}
	return items
}

func heldMessage(report ValidationReport) string {
	messages := make([]string, 0)
	for _, item := range slices.Concat(report.Compliance.Items, report.Structure.Items) {
		if item.Severity == content.SeverityError {
			messages = append(messages, item.Message)
		}
	}
	noun := " findings need"
	if len(messages) == 1 {
		noun = " finding needs"
	}
	return strconv.Itoa(len(messages)) + noun + " a decision: " + strings.Join(messages, "; ") +
		". Accept the page as it is or regenerate it."
}
