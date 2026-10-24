package main

import (
	"math"
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
		if input.Step == stepLinker && input.Counted == rowAt(t, rows, input.Stand, input.Path).NoBody {
			t.Errorf("%s %s %s counted %v, want a call over no body left out of the spread", input.Stand, input.Path, input.Step, input.Counted)
		}
		if input.Step == stepJudge && !input.Counted {
			t.Errorf("%s %s judge left out of the spread, want it counted over its stand-in body", input.Stand, input.Path)
		}
	}
	for _, step := range []string{stepWriter, stepWriterPack, stepLinker, stepJudge, stepJudgePack, stepMeta} {
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
	if len(budgets) != len(fitBudgets()) || budgets[4096] == 0 {
		t.Errorf("fits by budget = %v, want every page at each of %v", budgets, fitBudgets())
	}
	for _, budget := range fitBudgets() {
		if budgets[budget] != budgets[4096] {
			t.Errorf("%d pages at %d, %d at 4096; want every page at every budget", budgets[budget], budget, budgets[4096])
		}
	}
}

func TestTheEstimatedInputIsWithinAQuarterOfTheRealCount(t *testing.T) {
	t.Parallel()

	estimate, _ := measuredEstimate(t)
	measured := 0
	for _, input := range estimate.Inputs {
		if input.Step == stepWriterPack || input.Step == stepJudgePack {
			continue
		}
		measured++
		if off := math.Abs(float64(input.Estimated-input.Real)) / float64(input.Real); off > estimateTolerance {
			t.Errorf("%s %s %s estimates %d input tokens for a real %d, off by %.0f%%, over the %.0f%% allowed",
				input.Stand, input.Path, input.Step, input.Estimated, input.Real, 100*off, 100*estimateTolerance)
		}
	}
	if measured == 0 {
		t.Fatal("no step was measured")
	}
}

func TestFitRefusesNoPackThatFitsAndKeepsNoneOverItsBudget(t *testing.T) {
	t.Parallel()

	estimate, _ := measuredEstimate(t)
	tight := 0
	for _, fit := range estimate.Fits {
		switch {
		case fit.Whole <= 0:
			t.Errorf("%s %s has no whole real count", fit.Stand, fit.Path)
		case fit.Refused != "" && fit.Whole <= fit.Budget:
			t.Errorf("%s %s is refused at %d although its whole pack is %d real tokens: %s", fit.Stand, fit.Path, fit.Budget, fit.Whole, fit.Refused)
		case fit.Refused == "" && fit.Over:
			t.Errorf("%s %s kept %d real tokens at a budget of %d", fit.Stand, fit.Path, fit.Real, fit.Budget)
		}
		if fit.Whole > fit.Budget {
			tight++
		}
	}
	if tight == 0 {
		t.Error("no budget is tighter than a whole pack, so nothing proves a cut keeps under its budget")
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
	for _, want := range []string{"internal format", "input_tokens", "stand-in body"} {
		if !strings.Contains(notes, want) {
			t.Errorf("the notes never say %q:\n%s", want, notes)
		}
	}
	if len(estimate.Proposal) == 0 {
		t.Error("the report proposes nothing, not even that a constant holds")
	}
}
