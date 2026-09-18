package steps

import (
	"context"
	"encoding/json"
	"strconv"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

const (
	NameValidate     = "validate"
	ParamAllowErrors = "allowErrors"

	CodeTitleMissingKeyword = "primary_missing_in_title"
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

			links, _, err := run.Get[content.InsertResult](sc.Check, CheckpointLinks)
			if err != nil {
				return run.Result{}, err
			}

			report := ValidationReport{
				PageID:     sc.Page.ID,
				Compliance: content.Compliance(doc, lc, policy, sc.Page.ID),
				Structure:  content.Structure(doc, entity.PrimaryKeyword, entity.SecondaryKeywords, sc.Spec),
				Links:      links,
			}
			report.Structure.Items = append(report.Structure.Items, titleFindings(sc, entity.PrimaryKeyword)...)
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
			if !sc.BoolParam(ParamAllowErrors) && (report.Compliance.HasErrors() || report.Structure.HasErrors()) {
				return result, errors.New(errors.Invalid, "the page does not satisfy its template and its graph").
					WithDetail("pageId", sc.Page.ID).WithDetail("score", report.Score)
			}
			return result, nil
		},
	}
}

func titleFindings(sc *run.StepContext, primary string) []content.Finding {
	if !sc.Spec.KeywordRules.PrimaryInTitle || primary == "" {
		return nil
	}

	draft, err := sc.Artifact(run.ArtifactDraft)
	if err != nil {
		return nil
	}

	var decoded content.ContentDraft
	if unmarshalErr := json.Unmarshal(draft.Blob, &decoded); unmarshalErr != nil {
		return nil
	}
	if _, _, found := content.FindFold(decoded.Title, primary); found {
		return nil
	}

	return []content.Finding{{
		Severity: content.SeverityError, Code: CodeTitleMissingKeyword,
		Message: "the meta title does not carry the primary keyword",
		Details: map[string]any{"title": decoded.Title, "primaryKeyword": primary},
	}}
}
