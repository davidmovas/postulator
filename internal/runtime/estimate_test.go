package runtime_test

import (
	"context"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
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

	whole, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body", "generate_meta")),
		harness.specs.spec)
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	bodyOnly, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body")), harness.specs.spec)
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

	shallow := template.TemplateSpec{LinkRules: template.LinkRules{UpDepth: 1}}
	deep := template.TemplateSpec{LinkRules: template.LinkRules{UpDepth: 3}}

	one, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("repair_links")), shallow)
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	three, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("repair_links")), deep)
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if three.Tokens != one.Tokens*3 {
		t.Fatalf("three levels of parents cost %d against one level's %d", three.Tokens, one.Tokens)
	}

	none, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("repair_links")), template.TemplateSpec{})
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

	estimate, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body", "generate_images")),
		harness.specs.spec)
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if len(estimate.Findings) != 1 || estimate.Findings[0].Code != runtime.CodeUnpricedStep {
		t.Fatalf("findings = %+v, want the image step named as unpriced", estimate.Findings)
	}

	without, err := engine.EstimateRun(t.Context(), harness.newRun(recipeOf("generate_body")), harness.specs.spec)
	if err != nil {
		t.Fatalf("EstimateRun: %v", err)
	}
	if len(without.Findings) != 0 {
		t.Fatalf("findings = %+v, want none", without.Findings)
	}
}

func TestATokenBudgetPausesTheRun(t *testing.T) {
	t.Parallel()

	harness := newHarness(t, 2)
	harness.spend.set(llm.Spend{Usage: llm.Usage{Input: 900, Output: 600, Total: 1500}, USD: 0.02, Calls: 1})

	calls := newCounter()
	engine := harness.engine(t, mustRegister(t, bodyStep(calls), validateStep(calls)))

	record := harness.newRun(recipeOf("generate_body", "validate"))
	record.Budget = run.Budget{MaxTokens: 1000}
	queued, err := engine.Enqueue(t.Context(), record)
	if err != nil {
		t.Fatalf("Enqueue: %v", err)
	}

	paused := harness.waitForRun(t, queued.ID, run.StatusPaused)
	if paused.PauseReason != run.PauseBudgetExceeded {
		t.Fatalf("PauseReason = %q", paused.PauseReason)
	}
	if harness.bus.count(events.RunBudgetExceeded) != 0 {
		t.Fatalf("run.budget_exceeded names only money and was published %d times",
			harness.bus.count(events.RunBudgetExceeded))
	}
	if harness.bus.count(events.RunPaused) == 0 {
		t.Fatal("the run was never announced as paused")
	}
	assertGapless(t, harness, queued.ID)
}
