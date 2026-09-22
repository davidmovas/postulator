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

	judgeOutputTokens = 1024
)

type JudgeReport = appcontent.JudgeReport

func Judge(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameJudge,
		Role:     domainllm.RoleJudge,
		Requires: []run.ArtifactKind{run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactJudgeReport},
		Retry:    run.RetryPolicy{Max: 1},
		Price:    run.Price{OutputTokens: judgeOutputTokens},
		Run: func(ctx context.Context, sc *run.StepContext) (run.Result, error) {
			report, tokens, err := judgement(ctx, deps, sc)
			if err != nil {
				if ctx.Err() != nil {
					return run.Result{}, err
				}
				report = JudgeReport{
					Score:       0,
					Issues:      []string{"the judge could not be reached: " + err.Error()},
					Suggestions: []string{},
				}
			}

			blob, encodeErr := encode(report, "judge report")
			if encodeErr != nil {
				return run.Result{}, encodeErr
			}
			return run.Result{
				Artifacts: []run.Artifact{{Kind: run.ArtifactJudgeReport, Blob: blob}},
				Tokens:    tokens,
				Message:   "the judge scored " + strconv.FormatFloat(report.Score, 'f', 2, 64),
			}, nil
		},
	}
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

	assessed, err := deps.Content.Assess(ctx, appcontent.AssessRequest{
		SiteID: sc.Run.SiteID, Page: sc.Page, Entity: entity, Spec: sc.Spec, Body: doc.HTML(),
		Snippet:    appcontent.Snippet{Title: snippet.Title, Description: snippet.Description},
		HasSnippet: hasSnippet, Targets: targets, Call: callMeta(sc, NameJudge),
	})
	return assessed.Report, assessed.Tokens, err
}
