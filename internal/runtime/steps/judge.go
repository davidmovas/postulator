package steps

import (
	"context"
	"strconv"

	appcontent "github.com/davidmovas/postulator/internal/application/content"
	"github.com/davidmovas/postulator/internal/domain/content"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
)

const (
	NameJudge = appcontent.NameJudge

	CodeJudgeUnavailable = "judge_unavailable"
)

type JudgeReport = appcontent.JudgeReport

func Judge(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameJudge,
		Role:     domainllm.RoleJudge,
		Requires: []run.ArtifactKind{run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactJudgeReport},
		Retry:    run.RetryPolicy{Max: 1},
		Price:    run.Price{OutputTokens: appcontent.JudgeTokens},
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			report, tokens, err := judgement(ctx, deps, sc)
			if err != nil {
				if ctx.Err() != nil {
					return run.Result{}, err
				}
				report = unreachable(sc, err)
			}

			blob, encodeErr := encode(report, "judge report")
			if encodeErr != nil {
				return run.Result{}, encodeErr
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactJudgeReport, Blob: blob}},
				Tokens:    tokens,
				Message:   verdict(report, sc.Page.Path),
			}, nil
		},
	}
}

func unreachable(sc *run.StepContext, err error) JudgeReport {
	return JudgeReport{
		Issues:      []string{},
		Suggestions: []string{},
		Findings: []content.Finding{{
			Severity: content.SeverityWarn,
			Code:     CodeJudgeUnavailable,
			Message:  "the judge could not be reached, so " + sc.Page.Path + " carries no quality score: " + err.Error(),
			Details: map[string]any{
				"pageId": sc.Page.ID, "path": sc.Page.Path, "reason": err.Error(),
			},
		}},
	}
}

func verdict(report JudgeReport, path string) string {
	if report.Score == nil {
		return "the judge could not be reached, so " + path + " is not scored"
	}
	return "the judge scored " + strconv.FormatFloat(*report.Score, 'f', 2, 64)
}

func judgement(ctx context.Context, deps Deps, sc *run.StepContext) (report JudgeReport, tokens int, err error) {
	doc, err := bodyOf(sc)
	if err != nil {
		return JudgeReport{}, 0, err
	}
	entity, err := entityOf(ctx, deps, sc)
	if err != nil {
		return JudgeReport{}, 0, err
	}
	snippet, hasSnippet, err := decodeArtifact[Meta](sc, run.ArtifactMeta)
	if err != nil {
		return JudgeReport{}, 0, err
	}

	var targets []content.LinkTarget
	if lc, lcErr := linkContextOf(sc); lcErr == nil {
		targets = lc.Targets
	}

	body, err := doc.Render()
	if err != nil {
		return JudgeReport{}, 0, err
	}

	assessed, err := deps.Content.Assess(ctx, appcontent.AssessRequest{
		SiteID: sc.Run.SiteID, Page: sc.Page, Entity: entity, Spec: sc.Spec, Body: body,
		Snippet:    appcontent.Snippet{Title: snippet.Title, Description: snippet.Description},
		HasSnippet: hasSnippet, Targets: targets, Call: callMeta(sc, NameJudge),
	})
	return assessed.Report, assessed.Tokens, err
}
