package steps

import (
	"context"
	"strconv"
	"strings"

	port "github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/graph"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

const (
	NameJudge = "judge"

	judgeTokens = 1024
)

type JudgeReport struct {
	Score       float64  `json:"score" description:"The overall quality of the page between 0 and 1"`
	Issues      []string `json:"issues" description:"What is wrong with the page, worst first, one sentence each"`
	Suggestions []string `json:"suggestions" description:"What would raise the score, one sentence each"`
}

type judgePrompt struct {
	Page    pagemap.Page
	Entity  graph.Entity
	Spec    template.TemplateSpec
	Body    string
	Meta    Meta
	HasMeta bool
	Targets []content.LinkTarget
}

func Judge(deps Deps) run.StepDef {
	return run.StepDef{
		Name:     NameJudge,
		Role:     domainllm.RoleJudge,
		Requires: []run.ArtifactKind{run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactJudgeReport},
		Retry:    run.RetryPolicy{Max: 1},
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

func judgement(ctx context.Context, deps Deps, sc *run.StepContext) (JudgeReport, int, error) {
	doc, err := bodyOf(sc)
	if err != nil {
		return JudgeReport{}, 0, err
	}
	entity, err := entityOf(ctx, deps, sc)
	if err != nil {
		return JudgeReport{}, 0, err
	}
	ref, err := deps.Profiles.Resolve(ctx, sc.Run.SiteID, domainllm.RoleJudge, sc.Spec.ModelProfiles)
	if err != nil {
		return JudgeReport{}, 0, err
	}
	meta, hasMeta, err := decodeArtifact[Meta](sc, run.ArtifactMeta)
	if err != nil {
		return JudgeReport{}, 0, err
	}

	var targets []content.LinkTarget
	if lc, lcErr := linkContextOf(sc); lcErr == nil {
		targets = lc.Targets
	}

	system, user, err := render(NameJudge, judgePrompt{
		Page: sc.Page, Entity: entity, Spec: sc.Spec, Body: doc.HTML(),
		Meta: meta, HasMeta: hasMeta, Targets: targets,
	})
	if err != nil {
		return JudgeReport{}, 0, err
	}

	report, usage, err := port.Structured[JudgeReport](ctx, deps.LLM, port.Request{
		Ref:       ref,
		System:    system,
		Messages:  []port.Message{{Role: port.RoleUser, Text: user}},
		MaxTokens: judgeTokens,
		Meta:      callMeta(sc, NameJudge),
	})
	if err != nil {
		return JudgeReport{}, usage.Total, err
	}
	return settleJudgement(report), usage.Total, nil
}

func settleJudgement(report JudgeReport) JudgeReport {
	report.Score = min(max(report.Score, 0), 1)
	report.Issues = trimmed(report.Issues)
	report.Suggestions = trimmed(report.Suggestions)
	return report
}

func trimmed(lines []string) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if text := strings.TrimSpace(line); text != "" {
			out = append(out, text)
		}
	}
	return out
}
