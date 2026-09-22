package runs_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func enabledNames(recipe []template.StepSpec) []string {
	out := make([]string, 0, len(recipe))
	for _, step := range recipe {
		if step.Enabled {
			out = append(out, step.Name)
		}
	}
	return out
}

func startedWith(t *testing.T, fixture *fixture, req runs.StartRequest) []string {
	t.Helper()

	req.SiteID = fixture.siteID
	req.PageIDs = []string{fixture.pages[0]}
	if _, err := fixture.service.Start(t.Context(), req); err != nil {
		t.Fatalf("Start: %v", err)
	}
	return enabledNames(fixture.engine.queued.Recipe)
}

func TestAKindWithARecipeOfItsOwnNeverTakesTheTemplates(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		kind run.Kind
		want []string
	}{
		{
			name: "relink", kind: run.KindRelink,
			want: []string{"resolve_context", "relink_page", "sync_back", "report"},
		},
		{name: "repair", kind: run.KindRepair, want: []string{"repair_hierarchy", "sync_back", "report"}},
		{name: "sync", kind: run.KindSync, want: []string{"sync_site"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newFixture(t)
			fixture.specs.siteID = fixture.siteID

			got := startedWith(t, fixture, runs.StartRequest{Kind: string(tc.kind)})
			if !slices.Equal(got, tc.want) {
				t.Fatalf("a %s run runs %v, want %v", tc.kind, got, tc.want)
			}
			if fixture.engine.queued.Kind != tc.kind {
				t.Fatalf("the run was queued as %q", fixture.engine.queued.Kind)
			}
		})
	}
}

func TestARecipeThatDisagreesWithTheKindIsRefused(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.specs.siteID = fixture.siteID

	_, err := fixture.service.Start(t.Context(), runs.StartRequest{
		SiteID: fixture.siteID, PageIDs: []string{fixture.pages[0]}, Kind: string(run.KindRelink),
		Recipe: []template.StepSpec{{Name: "generate_body", Enabled: true}},
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Start with a recipe of its own = %v, want invalid", err)
	}
	if !strings.Contains(err.Error(), "relink") {
		t.Fatalf("the refusal does not name the kind: %v", err)
	}

	got := startedWith(t, fixture, runs.StartRequest{
		Kind: string(run.KindRelink), Recipe: run.RelinkRecipe(),
	})
	if !slices.Equal(got, enabledNames(run.RelinkRecipe())) {
		t.Fatalf("the kind's own recipe was refused: %v", got)
	}
}

func TestARevertIsNotStartedLikeAnyOtherRun(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.specs.siteID = fixture.siteID

	_, err := fixture.service.Start(t.Context(), runs.StartRequest{
		SiteID: fixture.siteID, PageIDs: []string{fixture.pages[0]}, Kind: string(run.KindRevert),
	})
	if !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Start of a revert = %v, want invalid", err)
	}
	if !strings.Contains(err.Error(), "run it undoes") {
		t.Fatalf("the refusal does not say where a revert comes from: %v", err)
	}
}

func TestAPageRunFallsBackToTheTemplateThenToTheGenerateRecipe(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.specs.siteID = fixture.siteID

	if got := startedWith(t, fixture, runs.StartRequest{}); !slices.Equal(got, []string{"generate_body"}) {
		t.Fatalf("the template recipe was not taken: %v", got)
	}

	fixture.specs.spec = template.TemplateSpec{}
	got := startedWith(t, fixture, runs.StartRequest{})
	if !slices.Equal(got, enabledNames(run.GenerateRecipe())) {
		t.Fatalf("a template with no recipe of its own runs %v, want the generate recipe", got)
	}
}

func TestAStepThatBelongsToAKindIsRefusedInAPageRecipe(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		kind    run.Kind
		refused bool
	}{
		{name: "a generate run", kind: run.KindGenerate, refused: true},
		{name: "an audit run", kind: run.KindAudit, refused: true},
		{name: "a custom run", kind: run.KindCustom},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newFixture(t)
			fixture.specs.siteID = fixture.siteID
			fixture.specs.spec = template.TemplateSpec{Recipe: []template.StepSpec{
				{Name: "resolve_context", Enabled: true},
				{Name: "repair_hierarchy", Enabled: true},
				{Name: "generate_body", Enabled: true},
			}}

			_, err := fixture.service.Start(t.Context(), runs.StartRequest{
				SiteID: fixture.siteID, PageIDs: []string{fixture.pages[0]}, Kind: string(tc.kind),
			})
			if !tc.refused {
				if err != nil {
					t.Fatalf("a custom run may name any step: %v", err)
				}
				return
			}
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Start = %v, want invalid", err)
			}
			if !strings.Contains(err.Error(), "repair_hierarchy") || !strings.Contains(err.Error(), "repair") {
				t.Fatalf("the refusal does not name the step and the kind that owns it: %v", err)
			}
		})
	}
}

func TestAStepThatIsNotEnabledIsNotJudged(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.specs.siteID = fixture.siteID
	fixture.specs.spec = template.TemplateSpec{Recipe: []template.StepSpec{
		{Name: "sync_site"},
		{Name: "generate_body", Enabled: true},
	}}

	if got := startedWith(t, fixture, runs.StartRequest{}); !slices.Equal(got, []string{"generate_body"}) {
		t.Fatalf("a disabled step was judged: %v", got)
	}
}
