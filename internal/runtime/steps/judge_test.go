package steps_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

const judgeBody = "<h1>Espresso guide</h1><p>Espresso is a way to make coffee.</p>"

func judgeContext(t *testing.T) *run.StepContext {
	t.Helper()

	return unitContext(t, map[run.ArtifactKind][]byte{
		run.ArtifactBodyHTML: []byte(judgeBody),
		run.ArtifactMeta:     []byte(`{"title":"espresso | Shop","description":"How to pull a shot."}`),
	})
}

func runJudge(t *testing.T, deps steps.Deps) steps.JudgeReport {
	t.Helper()

	result, err := steps.Judge(deps).Run(t.Context(), judgeContext(t))
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Kind != run.ArtifactJudgeReport {
		t.Fatalf("Judge produced %+v", result.Artifacts)
	}

	var report steps.JudgeReport
	if err = json.Unmarshal(result.Artifacts[0].Blob, &report); err != nil {
		t.Fatalf("decode the judge report: %v", err)
	}
	return report
}

func TestJudgeScoresThePage(t *testing.T) {
	t.Parallel()

	deps := judgeDeps(llmStub{reply: `{"score":0.8,"issues":["The body is thin."," "],"suggestions":["Add a worked example."]}`})

	report := runJudge(t, deps)
	if report.Score == nil || *report.Score != 0.8 {
		t.Errorf("score = %v, want 0.8", report.Score)
	}
	if len(report.Issues) != 1 || report.Issues[0] != "The body is thin." {
		t.Errorf("issues = %v, want the blank one dropped", report.Issues)
	}
	if len(report.Suggestions) != 1 {
		t.Errorf("suggestions = %v", report.Suggestions)
	}
}

func TestJudgeClampsTheScore(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		reply string
		want  float64
	}{
		{name: "above one", reply: `{"score":7}`, want: 1},
		{name: "below zero", reply: `{"score":-3}`, want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := judgeDeps(llmStub{reply: tc.reply})
			got := runJudge(t, deps).Score
			if got == nil || *got != tc.want {
				t.Fatalf("score = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestJudgeNeverFailsTheItemAndNeverScoresItEither(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{name: "the provider is down", err: errors.New(errors.External, "the judge is unreachable")},
		{name: "the provider refuses the key", err: errors.New(errors.Unauthorized, "the api key is rejected")},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			report := runJudge(t, judgeDeps(llmStub{err: tc.err}))
			if report.Score != nil {
				t.Errorf("score = %v, want none: an unreachable judge scores nothing", *report.Score)
			}
			if len(report.Issues) != 0 {
				t.Errorf("issues = %v, want none: the model said nothing about the page", report.Issues)
			}
			if len(report.Findings) != 1 {
				t.Fatalf("findings = %+v, want the one that says the judge was not reached", report.Findings)
			}
			finding := report.Findings[0]
			if finding.Code != steps.CodeJudgeUnavailable || finding.Severity != content.SeverityWarn {
				t.Errorf("finding = %+v", finding)
			}
			if !strings.Contains(finding.Message, tc.err.Error()) {
				t.Errorf("message = %q, want the provider's own words", finding.Message)
			}
			if finding.Details["pageId"] != "page-child" {
				t.Errorf("details = %+v, want the page named", finding.Details)
			}
		})
	}
}

func TestJudgeReportsAnUnscoredPageInTheFinalReport(t *testing.T) {
	t.Parallel()

	result, err := steps.Judge(judgeDeps(llmStub{err: errors.New(errors.External, "the judge is unreachable")})).
		Run(t.Context(), judgeContext(t))
	if err != nil {
		t.Fatalf("Judge: %v", err)
	}

	final := runReport(t, map[run.ArtifactKind][]byte{
		run.ArtifactValidationReport: []byte(storedValidation),
		run.ArtifactJudgeReport:      result.Artifacts[0].Blob,
	})
	if final.Score == nil || *final.Score != 0.7 {
		t.Fatalf("score = %v, want the validation score untouched by a judge that never ran", final.Score)
	}
	if final.Warnings != 2 {
		t.Errorf("warnings = %d, want the structure warning and the judge that could not be reached", final.Warnings)
	}
}

func TestJudgeStillReportsACancelledContext(t *testing.T) {
	t.Parallel()

	deps := judgeDeps(llmStub{err: errors.New(errors.Cancelled, "the call was cancelled")})

	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	if _, err := steps.Judge(deps).Run(ctx, judgeContext(t)); !errors.IsCode(err, errors.Cancelled) {
		t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), errors.Cancelled, err)
	}
}

func TestJudgeCarriesTheRubricAndTheSnippet(t *testing.T) {
	t.Parallel()

	recorder := &promptRecorder{reply: `{"score":1}`}
	deps := judgeDeps(recorder)

	runJudge(t, deps)
	for _, want := range []string{"RUBRIC", "espresso | Shop", judgeBody} {
		if !strings.Contains(recorder.last, want) {
			t.Errorf("the prompt does not carry %q:\n%s", want, recorder.last)
		}
	}
}
