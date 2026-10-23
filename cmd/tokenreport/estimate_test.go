package main

import (
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/llm/catalog"
)

func measuredEstimate(t *testing.T) (estimateReport, []pageRow) {
	t.Helper()

	models, err := catalog.New(noOverrides{})
	if err != nil {
		t.Fatalf("catalog: %v", err)
	}
	held := newCounters()
	rows, err := measurePages(t.Context(), held, models)
	if err != nil {
		t.Fatalf("measurePages: %v", err)
	}
	estimate, err := measureEstimate(t.Context(), held, models, rows)
	if err != nil {
		t.Fatalf("measureEstimate: %v", err)
	}
	return estimate, rows
}

func TestEveryMeasuredCallStandsBesideItsEstimate(t *testing.T) {
	t.Parallel()

	estimate, rows := measuredEstimate(t)
	steps := map[string]int{}
	for _, input := range estimate.Inputs {
		steps[input.Step]++
		if input.Estimated <= 0 || input.Real <= 0 {
			t.Errorf("%s %s %s = %+v, want both counts", input.Stand, input.Path, input.Step, input)
		}
		if want := float64(input.Real) / float64(input.Estimated); input.Ratio != want {
			t.Errorf("%s %s %s ratio = %v, want real over estimated, %v", input.Stand, input.Path, input.Step, input.Ratio, want)
		}
		if (input.Step == stepJudge || input.Step == stepLinker) && input.Counted == rowAt(t, rows, input.Stand, input.Path).NoBody {
			t.Errorf("%s %s %s counted %v, want a call over no body left out of the spread", input.Stand, input.Path, input.Step, input.Counted)
		}
	}
	for _, step := range []string{stepWriter, stepWriterPack, stepLinker, stepJudge, stepJudgePack} {
		if steps[step] != 8 {
			t.Errorf("%d %s rows, want one per measured page", steps[step], step)
		}
	}
}

func TestFitIsCountedAtEachBudget(t *testing.T) {
	t.Parallel()

	estimate, _ := measuredEstimate(t)
	budgets := map[int]int{}
	for _, fit := range estimate.Fits {
		budgets[fit.Budget]++
		if fit.Refused != "" {
			continue
		}
		if fit.Real <= 0 || fit.Over != (fit.Real > fit.Budget) {
			t.Errorf("%s %s at %d = %+v, want a real count and over when it passes the budget", fit.Stand, fit.Path, fit.Budget, fit)
		}
	}
	if len(budgets) != 3 || budgets[4096] == 0 || budgets[4096] != budgets[3000] || budgets[3000] != budgets[1500] {
		t.Errorf("fits by budget = %v, want every page at 4096, 3000 and 1500", budgets)
	}
}

func TestOnlyProseCountsTowardsTokensPerWord(t *testing.T) {
	t.Parallel()

	estimate, _ := measuredEstimate(t)
	if len(estimate.Words) == 0 {
		t.Fatal("no body was long enough to count its tokens per word")
	}
	seen := map[string]bool{}
	for _, words := range estimate.Words {
		if words.Words < minProseWords {
			t.Errorf("%s %s has %d words, under the %d a sample needs", words.Stand, words.Path, words.Words, minProseWords)
		}
		if seen[words.Path] {
			t.Errorf("%s is counted twice; a body both stands carry is one sample", words.Path)
		}
		seen[words.Path] = true
		if words.Text <= 0 || words.HTML < words.Text || words.Escaped < words.HTML {
			t.Errorf("%s counts text %d, HTML %d, escaped %d; want each at least the one before", words.Path, words.Text, words.HTML, words.Escaped)
		}
	}
}

func TestTheReportStatesItsSpreadsAndFootnotes(t *testing.T) {
	t.Parallel()

	estimate, _ := measuredEstimate(t)
	named := map[string]bool{}
	for _, s := range estimate.Spreads {
		named[s.Name] = true
		if s.Count == 0 || s.Min > s.Median || s.Median > s.Max {
			t.Errorf("spread %+v is out of order or empty", s)
		}
	}
	for _, want := range []string{"writer input", "pack", "section", "Fit at 1500", "body text"} {
		found := false
		for name := range named {
			found = found || strings.HasPrefix(name, want)
		}
		if !found {
			t.Errorf("no spread starts with %q in %v", want, named)
		}
	}
	notes := strings.Join(estimate.Notes, "\n")
	for _, want := range []string{"internal format", "input_tokens", "no body term"} {
		if !strings.Contains(notes, want) {
			t.Errorf("the notes never say %q:\n%s", want, notes)
		}
	}
	if len(estimate.Proposal) == 0 {
		t.Error("the report proposes nothing, not even that a constant holds")
	}
}
