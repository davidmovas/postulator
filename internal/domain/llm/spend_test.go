package llm_test

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
)

func TestPurposeOf(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		runID string
		step  string
		want  llm.Purpose
	}{
		{name: "a step of a run", runID: "run-1", step: "generate_body", want: llm.PurposeRun},
		{name: "the judge inside a run", runID: "run-1", step: llm.StepJudge, want: llm.PurposeRun},
		{name: "a run step with no step recorded", runID: "run-1", want: llm.PurposeRun},
		{name: "an agent round", step: llm.StepChat, want: llm.PurposeChat},
		{name: "a conversation title", step: llm.StepTitle, want: llm.PurposeTitle},
		{name: "a provider test", step: llm.StepProbe, want: llm.PurposeProbe},
		{name: "pages proposed as entities", step: llm.StepProposeFromPages, want: llm.PurposeGraph},
		{name: "keywords proposed as entities", step: llm.StepProposeFromKeywords, want: llm.PurposeGraph},
		{name: "related edges proposed", step: llm.StepProposeRelated, want: llm.PurposeGraph},
		{name: "a page judged on demand", step: llm.StepJudge, want: llm.PurposeAudit},
		{name: "a call that names no step", want: llm.PurposeOther},
		{name: "a step no rule knows", step: "generate_images", want: llm.PurposeOther},
		{name: "a step name in another case", step: "CHAT", want: llm.PurposeOther},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := llm.PurposeOf(tc.runID, tc.step); got != tc.want {
				t.Errorf("PurposeOf(%q, %q) = %q, want %q", tc.runID, tc.step, got, tc.want)
			}
		})
	}
}

func TestPurposeRulesNameEachStepOnce(t *testing.T) {
	t.Parallel()

	seen := map[string]llm.Purpose{}
	purposes := map[llm.Purpose]bool{}
	for _, rule := range llm.PurposeRules() {
		if rule.Purpose == llm.PurposeRun || rule.Purpose == llm.PurposeOther {
			t.Errorf("%q is decided by the run and the fallback, not by a step", rule.Purpose)
		}
		if purposes[rule.Purpose] {
			t.Errorf("%q has two rules", rule.Purpose)
		}
		purposes[rule.Purpose] = true
		if len(rule.Steps) == 0 {
			t.Errorf("%q names no step", rule.Purpose)
		}
		for _, step := range rule.Steps {
			if owner, taken := seen[step]; taken {
				t.Errorf("%q belongs to %q and %q", step, owner, rule.Purpose)
			}
			seen[step] = rule.Purpose
			if got := llm.PurposeOf("", step); got != rule.Purpose {
				t.Errorf("PurposeOf(%q) = %q, want the rule's %q", step, got, rule.Purpose)
			}
		}
	}
}

func TestTheJudgeStepIsTheRunsJudge(t *testing.T) {
	t.Parallel()

	if llm.StepJudge != string(run.StepJudge) {
		t.Fatalf("llm.StepJudge = %q, want the run's judge step %q", llm.StepJudge, run.StepJudge)
	}
}
