package runtime_test

import (
	"context"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/content"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime"
)

func pricedStep(name string, role llm.Role, price run.Price) run.StepDef {
	return run.StepDef{
		Name:     name,
		Role:     role,
		Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Price:    price,
		Run:      func(context.Context, *run.StepContext) (run.Result, error) { return run.Result{}, nil },
	}
}

func TestEstimatePricesAStepAtTheCeilingItDeclares(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	small := pricedStep("generate_meta", llm.RoleEditor, run.Price{OutputTokens: 512})
	body := pricedStep("generate_body", llm.RoleWriter, run.Price{})
	engine := harness.engine(t, mustRegister(t, body, small))

	harness.specs.spec = template.TemplateSpec{
		Sections: []template.Section{{Heading: "Intro", TargetWords: 3000}},
	}

	whole, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body", "generate_meta")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	bodyOnly, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("EstimateRun of the body alone: %v", err)
	}

	meta := whole.Tokens - bodyOnly.Tokens
	if meta >= bodyOnly.Tokens {
		t.Fatalf("the capped step is priced at %d tokens against the body's %d; it must be priced at its own ceiling",
			meta, bodyOnly.Tokens)
	}
	if meta <= 512 {
		t.Fatalf("the capped step is priced at %d tokens, want its 512 output tokens and the prompt around them", meta)
	}
}

func TestEstimatePricesOneCallPerRepairIteration(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	repair := pricedStep("repair_links", llm.RoleLinker, run.Price{
		OutputTokens: 256,
		Calls: func(spec template.TemplateSpec, params map[string]any) int {
			return spec.LinkRules.UpDepth * 2
		},
	})
	engine := harness.engine(t, mustRegister(t, repair))

	harness.specs.spec = template.TemplateSpec{LinkRules: template.LinkRules{UpDepth: 1}}
	one, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("repair_links")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	harness.specs.spec = template.TemplateSpec{LinkRules: template.LinkRules{UpDepth: 3}}
	three, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("repair_links")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if three.Tokens != one.Tokens*3 {
		t.Fatalf("three levels of parents cost %d against one level's %d", three.Tokens, one.Tokens)
	}

	harness.specs.spec = template.TemplateSpec{}
	none, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("repair_links")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if none.Tokens != 0 || none.USD != 0 {
		t.Fatalf("a template that asks for no parent links costs %+v", none)
	}
}

func TestEstimateNamesTheStepItCannotPrice(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	body := pricedStep("generate_body", llm.RoleWriter, run.Price{})
	images := run.StepDef{
		Name:     "generate_images",
		Requires: []run.ArtifactKind{run.ArtifactBodyHTML},
		Produces: []run.ArtifactKind{run.ArtifactImages},
		Price:    run.Price{Unpriced: true},
		Run:      func(context.Context, *run.StepContext) (run.Result, error) { return run.Result{}, nil },
	}
	engine := harness.engine(t, mustRegister(t, body, images))

	estimate, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body", "generate_images")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if len(estimate.Findings) != 1 || estimate.Findings[0].Code != runtime.CodeUnpricedStep {
		t.Fatalf("findings = %+v, want the image step named as unpriced", estimate.Findings)
	}
	if estimate.Findings[0].Severity != content.SeverityWarn {
		t.Fatalf("an unpriced step is graded %q, want a warning", estimate.Findings[0].Severity)
	}

	without, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if len(without.Findings) != 0 {
		t.Fatalf("findings = %+v, want none", without.Findings)
	}
}

func TestARevertRunIsPricedAtNothing(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	reverting := run.StepDef{
		Name:     string(run.StepRevert),
		Produces: []run.ArtifactKind{run.ArtifactRevertResult},
		Run:      func(context.Context, *run.StepContext) (run.Result, error) { return run.Result{}, nil },
	}
	engine := harness.engine(t, mustRegister(t, reverting))

	record := harness.newRun(run.RevertRecipe())
	record.Kind = run.KindRevert

	estimate, err := engine.EstimateRun(t.Context(), record)
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if estimate.Tokens != 0 || estimate.USD != 0 || len(estimate.Findings) != 0 {
		t.Fatalf("a revert is priced at %+v, want nothing", estimate)
	}
}

func TestATokenBudgetPausesTheRun(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	harness.spend.set(llm.Spend{Usage: llm.Usage{Input: 30, Output: 20, Total: 50}, USD: 0.02, Calls: 1})

	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, bodyStep(calls), validateStep(calls)))

	record := harness.newRun(recipeOf("generate_body", "validate"))
	record.Budget = run.Budget{MaxTokens: 10}
	queued, err := engine.Enqueue(t.Context(), record)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	paused := harness.waitForRun(t, queued.ID, run.StatusPaused)
	if paused.PauseReason != run.PauseBudgetExceeded {
		t.Fatalf("PauseReason = %q", paused.PauseReason)
	}
	if harness.bus.count(events.RunPaused) == 0 {
		t.Fatal("the run was never announced as paused")
	}

	said := budgetExceeded(t, harness)
	if said.RunID != queued.ID {
		t.Fatalf("the event names the run %q, want %q", said.RunID, queued.ID)
	}
	if said.SpentTokens != 50 || said.BudgetTokens != 10 {
		t.Fatalf("the event says %d of %d tokens, want 50 of 10", said.SpentTokens, said.BudgetTokens)
	}
	if said.BudgetUSD != 0 {
		t.Fatalf("the event names a money cap of %v, want none", said.BudgetUSD)
	}
	assertGapless(t, harness, queued.ID)
}

func TestEstimateSumsEveryTargetOnItsOwnTemplate(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	body := pricedStep("generate_body", llm.RoleWriter, run.Price{})
	engine := harness.engine(t, mustRegister(t, body))

	short := template.TemplateSpec{Sections: []template.Section{{Heading: "Intro", TargetWords: 300}}}
	long := template.TemplateSpec{Sections: []template.Section{{Heading: "Intro", TargetWords: 3000}}}
	harness.specs.perPage = map[string]template.TemplateSpec{harness.pages[0]: short, harness.pages[1]: long}

	both, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	harness.pages = harness.pages[:1]
	first, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("EstimateRun of the short page: %v", err)
	}
	if both.Tokens <= first.Tokens*2 {
		t.Fatalf("two pages on different templates are priced at %d against %d for the short one alone; "+
			"the long one must be priced on its own template", both.Tokens, first.Tokens)
	}
	if len(both.Findings) != 0 {
		t.Fatalf("findings = %+v, want none", both.Findings)
	}
}

func TestEstimateReportsWhatWouldStopTheRunBeforeItStarts(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		arrange  func(h *harness)
		code     string
		severity content.Severity
	}{
		{
			name:     "the provider of the writer holds no key",
			arrange:  func(h *harness) { h.keys.missing = llm.SecretRef("openai") },
			code:     runtime.CodeProviderKeyMissing,
			severity: content.SeverityError,
		},
		{
			name:     "no profile answers for the role",
			arrange:  func(h *harness) { h.profiles.err = errors.New(errors.NotFound, "no profile for the writer") },
			code:     runtime.CodeModelUnresolved,
			severity: content.SeverityError,
		},
		{
			name:     "the catalog does not know the model",
			arrange:  func(h *harness) { h.catalog.err = errors.New(errors.NotFound, "no such model") },
			code:     runtime.CodeModelUnknown,
			severity: content.SeverityWarn,
		},
		{
			name: "the template of a page names another recipe",
			arrange: func(h *harness) {
				h.specs.spec.Recipe = recipeOf("generate_body", "validate")
			},
			code:     runtime.CodeRecipeDiffers,
			severity: content.SeverityWarn,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			harness := newHarness(t, 1)
			tc.arrange(harness)
			body := pricedStep("generate_body", llm.RoleWriter, run.Price{})
			engine := harness.engine(t, mustRegister(t, body))

			estimate, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body")))
			if err != nil {
				t.Fatalf("EstimateRun: %v", err)
			}
			if len(estimate.Findings) != 1 || estimate.Findings[0].Code != tc.code || estimate.Findings[0].Severity != tc.severity {
				t.Fatalf("findings = %+v, want one %s graded %s", estimate.Findings, tc.code, tc.severity)
			}
			if tc.code == runtime.CodeRecipeDiffers && estimate.Findings[0].PageID != harness.pages[0] {
				t.Fatalf("the finding names the page %q, want %s", estimate.Findings[0].PageID, harness.pages[0])
			}
			if blocking := len(estimate.Blocking()); (tc.severity == content.SeverityError) != (blocking == 1) {
				t.Fatalf("Blocking = %d findings for a %s", blocking, tc.severity)
			}
		})
	}
}

func TestEstimateRunsThePreflightOfEveryStepOnce(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	seen := 0
	checked := run.StepDef{
		Name:     "generate_body",
		Produces: []run.ArtifactKind{run.ArtifactBodyHTML},
		Run:      func(context.Context, *run.StepContext) (run.Result, error) { return run.Result{}, nil },
		Preflight: func(_ context.Context, record run.Run, targets map[string]run.Target) ([]run.EstimateFinding, error) {
			seen++
			out := make([]run.EstimateFinding, 0, len(targets))
			for _, id := range record.Targets {
				target := targets[id]
				out = append(out, run.EstimateFinding{
					Severity: content.SeverityError, Code: "entity_missing", PageID: target.Page.ID, Path: target.Page.Path,
					Message: target.Page.Path + " is mapped to nothing",
				})
			}
			return out, nil
		},
	}
	engine := harness.engine(t, mustRegister(t, checked))

	estimate, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if seen != 1 {
		t.Fatalf("the preflight ran %d times, want once for the whole run", seen)
	}
	if len(estimate.Findings) != 2 || len(estimate.Blocking()) != 2 {
		t.Fatalf("findings = %+v, want one blocking finding per page", estimate.Findings)
	}
	if estimate.Findings[0].Path != "/page-a/" || estimate.Findings[1].Path != "/page-b/" {
		t.Fatalf("findings = %+v, want them in the order of the targets", estimate.Findings)
	}
}

func TestEstimatePricesImagesOnTheModelThatDrawsThem(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 1)
	drawer := llm.ModelRef{Provider: "openai", Model: "gpt-image-2"}
	images := run.StepDef{
		Name:     "generate_images",
		Produces: []run.ArtifactKind{run.ArtifactImages},
		Run:      func(context.Context, *run.StepContext) (run.Result, error) { return run.Result{}, nil },
		Price: run.Price{
			Ref: &drawer, OutputTokens: 1000,
			Calls: func(spec template.TemplateSpec, _ map[string]any) int { return spec.Images.Inline },
		},
	}
	engine := harness.engine(t, mustRegister(t, images))

	harness.specs.spec = template.TemplateSpec{Images: template.Images{Inline: 3, Source: template.ImagesAI}}
	three, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_images")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	harness.specs.spec = template.TemplateSpec{Images: template.Images{Inline: 1, Source: template.ImagesAI}}
	one, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_images")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if one.Tokens == 0 || three.Tokens != one.Tokens*3 || three.USD != one.USD*3 {
		t.Fatalf("three images cost %+v against one image's %+v", three, one)
	}
	if len(three.Findings) != 0 {
		t.Fatalf("findings = %+v, want none", three.Findings)
	}

	harness.keys.missing = llm.SecretRef(drawer.Provider)
	unkeyed, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_images")))
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if len(unkeyed.Blocking()) != 1 || unkeyed.Blocking()[0].Code != runtime.CodeProviderKeyMissing {
		t.Fatalf("findings = %+v, want the missing key of the image model", unkeyed.Findings)
	}
}
