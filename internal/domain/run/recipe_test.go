package run_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
)

func names(recipe []template.StepSpec) []string {
	out := make([]string, 0, len(recipe))
	for _, step := range recipe {
		if step.Enabled {
			out = append(out, step.Name)
		}
	}
	return out
}

func TestAKindThatOwnsARecipeHandsItOut(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		kind  run.Kind
		want  []string
		owned bool
	}{
		{
			name: "relink", kind: run.KindRelink, owned: true,
			want: []string{"resolve_context", "relink_page", "sync_back", "report"},
		},
		{
			name: "repair", kind: run.KindRepair, owned: true,
			want: []string{"repair_hierarchy", "sync_back", "report"},
		},
		{name: "sync", kind: run.KindSync, owned: true, want: []string{"sync_site"}},
		{name: "revert", kind: run.KindRevert, owned: true, want: []string{"revert"}},
		{name: "generate", kind: run.KindGenerate},
		{name: "custom", kind: run.KindCustom},
		{name: "audit", kind: run.KindAudit},
		{name: "import", kind: run.KindImport},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			recipe, owned := tc.kind.Recipe()
			if owned != tc.owned {
				t.Fatalf("%s owns a recipe = %t, want %t", tc.kind, owned, tc.owned)
			}
			if !owned {
				if recipe != nil {
					t.Fatalf("%s hands out %v while owning no recipe", tc.kind, names(recipe))
				}
				return
			}
			if got := names(recipe); !slices.Equal(got, tc.want) {
				t.Fatalf("%s recipe = %v, want %v", tc.kind, got, tc.want)
			}
		})
	}
}

func TestTheGenerateRecipeWritesAndPublishesOnePage(t *testing.T) {
	t.Parallel()

	want := []string{
		"resolve_context", "generate_body", "generate_meta", "insert_links", "repair_links",
		"generate_images", "validate", "judge", "publish", "relink_neighbors", "sync_back", "report",
	}
	if got := names(run.GenerateRecipe()); !slices.Equal(got, want) {
		t.Fatalf("GenerateRecipe() = %v, want %v", got, want)
	}
	for _, step := range run.GenerateRecipe() {
		if run.PerKindStep(step.Name) {
			t.Errorf("the generate recipe names %s, which belongs to a kind of its own", step.Name)
		}
	}
}

func TestAStepBelongsToATemplateRecipeOrToAKindNeverBoth(t *testing.T) {
	t.Parallel()

	perKind := run.PerKindStepNames()
	if len(perKind) != 4 {
		t.Fatalf("PerKindStepNames() = %v, want the four steps a kind recipe owns", perKind)
	}

	for _, name := range perKind {
		if !run.PerKindStep(string(name)) {
			t.Errorf("%s is named by a kind recipe and is still offered to a template", name)
		}
		if slices.Contains(run.StepNames(), run.StepName(name)) {
			t.Errorf("%s is a step name, so a blank template recipe would carry it", name)
		}
	}
	for _, name := range run.StepNames() {
		if run.PerKindStep(string(name)) {
			t.Errorf("a template recipe is offered %s, which only a kind may name", name)
		}
	}
	if run.PerKindStep("publish") {
		t.Error("publish is a step every template recipe may name")
	}
}

func TestEveryPerKindStepIsNamedByExactlyOneKind(t *testing.T) {
	t.Parallel()

	kinds := []run.Kind{
		run.KindGenerate, run.KindRelink, run.KindAudit, run.KindSync, run.KindImport,
		run.KindRepair, run.KindRevert, run.KindCustom,
	}

	for _, step := range []string{"repair_hierarchy", "sync_site", "relink_page", "revert"} {
		owners := make([]run.Kind, 0, 1)
		for _, kind := range kinds {
			recipe, owned := kind.Recipe()
			if owned && slices.Contains(names(recipe), step) {
				owners = append(owners, kind)
			}
		}
		if len(owners) != 1 {
			t.Errorf("%s is named by %v, want exactly one kind", step, owners)
		}
	}
}
