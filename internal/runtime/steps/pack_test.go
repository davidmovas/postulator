package steps_test

import (
	stderrors "errors"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/contextpack"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func stubOf(t *testing.T, deps steps.Deps) *pictureStub {
	t.Helper()

	stub, held := deps.Pictures.(*pictureStub)
	if !held {
		t.Fatalf("the deps read pictures from %T, not the stub", deps.Pictures)
	}
	return stub
}

func childTarget() run.Target {
	return run.Target{
		Page: pagemap.Page{ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", WPType: pagemap.WPPage, EntityID: pointer("child")},
		Spec: spec(), Links: unitLinks(),
	}
}

func TestThePackReadingStepsPriceTheirPack(t *testing.T) {
	t.Parallel()

	priced := run.NewRegistry()
	if err := steps.Register(priced, unitDeps()); err != nil {
		t.Fatalf("Register: %v", err)
	}
	bare := run.NewRegistry()
	if err := steps.Register(bare, steps.Deps{}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	for _, name := range []string{steps.NameGenerateBody, steps.NameGenerateMeta, steps.NameJudge} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			def, _ := priced.Lookup(name)
			if def.Price.Input == nil {
				t.Fatalf("%s prices no input", name)
			}
			tokens, err := def.Price.Input(t.Context(), childTarget())
			if err != nil || tokens <= 0 {
				t.Errorf("%s prices its pack at %d tokens, %v; want the pack's size", name, tokens, err)
			}
			if def, _ := bare.Lookup(name); def.Price.Input != nil {
				t.Errorf("%s prices a pack it has no read layer for", name)
			}
		})
	}
}

func TestTheWriterPromptCarriesThePicture(t *testing.T) {
	t.Parallel()

	deps := unitDeps()
	deps.Pages = pageList{items: []pagemap.Page{
		{ID: "page-parent", SiteID: "site", Path: "/coffee/", Title: "Coffee", WPType: pagemap.WPPage, Status: pagemap.StatusPublished, EntityID: pointer("parent")},
		{ID: "page-child", SiteID: "site", Path: "/coffee/espresso/", Title: "Espresso", WPType: pagemap.WPPage, Status: pagemap.StatusPlanned,
			EntityID: pointer("child"), ParentPageID: pointer("page-parent"), ParentReason: pagemap.ParentWorkbook},
	}}
	deps = pictured(deps)
	recorder := &promptRecorder{reply: goodDraft}
	deps.LLM = recorder
	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})

	if _, err := steps.GenerateBody(deps).Run(t.Context(), sc); err != nil {
		t.Fatalf("GenerateBody: %v", err)
	}
	for _, want := range []string{
		"THE FULL PICTURE\n",
		"THE PAGE\nPath: /coffee/espresso/\nTitle: Espresso\n",
		"THE FIELD'S RULES\nField: Body\n",
		"THE TRAIL, NEAREST PARENT FIRST\nThe parent comes from the workbook.\n- Coffee (/coffee/",
		"PAGES THIS FIELD LINKS TO\n- /coffee/ (Coffee, above it, required",
		"WHAT TO WRITE\n",
	} {
		if !strings.Contains(recorder.last, want) {
			t.Errorf("the prompt lacks %q:\n%s", want, recorder.last)
		}
	}
	if strings.Contains(recorder.last, "KEYWORDS, BY VOLUME") {
		t.Errorf("the body prompt repeats the pack's keyword list beside its own:\n%s", recorder.last)
	}
	asked := stubOf(t, deps).requests()
	if len(asked) != 1 || asked[0].Claim == nil || asked[0].Claim.Links == nil || asked[0].Field != "body" {
		t.Errorf("the writer asked for %+v, want one pack over the claim's link context", asked)
	}
}

func TestTheWriterPackIsFittedToTheInputBudget(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		budget int
		window int
		check  func(t *testing.T, asked int)
	}{
		{name: "the model's room less what the call carries", window: 400000, check: func(t *testing.T, asked int) {
			if asked <= 300000 || asked >= 400000-4096 {
				t.Errorf("the default budget = %d, want the window less the ceiling, the reasoning and the prompt", asked)
			}
		}},
		{name: "a budget the person set", budget: 5000, check: func(t *testing.T, asked int) {
			if asked != 5000 {
				t.Errorf("the budget = %d, want the setting's 5000", asked)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			deps.Tuning = tuningStub{budget: tc.budget}
			deps.Models = catalogStub{context: tc.window}
			deps.LLM = &promptRecorder{reply: goodDraft}
			sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
			if _, err := steps.GenerateBody(deps).Run(t.Context(), sc); err != nil {
				t.Fatalf("GenerateBody: %v", err)
			}
			asked := stubOf(t, deps).requests()
			tc.check(t, asked[0].Budget)
		})
	}
}

func TestAnUndersizedInputBudgetIsRefusedBeforeTheRun(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		budget int
		codes  []string
	}{
		{name: "no budget set", codes: []string{}},
		{name: "a budget that fits", budget: 50000, codes: []string{}},
		{name: "a budget below the page and the rules", budget: 5, codes: []string{steps.CodeInputBudgetTooSmall}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			deps := unitDeps()
			deps.Tuning = tuningStub{budget: tc.budget}
			record, targets := preflightRun(nil, childTarget().Page)
			findings, err := steps.GenerateBody(deps).Preflight(t.Context(), record, targets)
			if err != nil {
				t.Fatalf("Preflight: %v", err)
			}
			if got := codesOf(findings); len(got) != len(tc.codes) || (len(got) > 0 && got[0] != tc.codes[0]) {
				t.Fatalf("findings = %+v, want %v", findings, tc.codes)
			}
			if len(findings) > 0 && (findings[0].PageID != "page-child" || !strings.Contains(findings[0].Message, "llm.inputBudget.writer")) {
				t.Errorf("finding = %+v, want it naming the page and the setting", findings[0])
			}
		})
	}

	deps := unitDeps()
	deps.Tuning = tuningStub{budget: 5}
	sc := unitContext(t, map[run.ArtifactKind][]byte{run.ArtifactLinkContext: linkContextBlob(t, deps)})
	_, err := steps.GenerateBody(deps).Run(t.Context(), sc)
	refused, typed := stderrors.AsType[*errors.Error](err)
	if !typed || !errors.IsCode(err, errors.Invalid) || refused.Details["reason"] != contextpack.ReasonBudgetTooSmall {
		t.Errorf("a run past the preflight = %v, want the budget refused", err)
	}
}
