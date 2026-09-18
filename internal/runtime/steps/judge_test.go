package steps_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

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
	if report.Score != 0.8 {
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
			if got := runJudge(t, deps).Score; got != tc.want {
				t.Fatalf("score = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestJudgeNeverFailsTheItem(t *testing.T) {
	t.Parallel()

	deps := judgeDeps(llmStub{err: errors.New(errors.External, "the judge is unreachable")})

	report := runJudge(t, deps)
	if report.Score != 0 {
		t.Errorf("score = %v, want 0 when the judge could not be reached", report.Score)
	}
	if len(report.Issues) != 1 || !strings.Contains(report.Issues[0], "could not be reached") {
		t.Fatalf("issues = %v", report.Issues)
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
