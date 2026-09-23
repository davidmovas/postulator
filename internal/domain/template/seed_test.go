package template_test

import (
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/domain/template"
)

func TestSeedShipsFiveValidGlobalTemplates(t *testing.T) {
	t.Parallel()

	seeds := template.Seed()
	names := make([]string, 0, len(seeds))
	for i := range seeds {
		seed := &seeds[i]
		names = append(names, seed.Name)
		if seed.Scope != template.ScopeGlobal || seed.SiteID != nil || seed.Version != 1 || seed.ID != "" {
			t.Errorf("%s: seed must be global, unversioned beyond 1 and carry no id, got %+v", seed.Name, seed)
		}
		if err := template.Validate(seed.Spec); err != nil {
			t.Errorf("%s: %v", seed.Name, err)
		}
		if seed.PageKind == "" || len(seed.Spec.Recipe) == 0 || seed.Spec.Length.Min == 0 || seed.Spec.MetaRules.DescriptionMax == 0 {
			t.Errorf("%s: seed is missing content: %+v", seed.Name, seed.Spec)
		}
		if len(seed.Spec.ModelProfiles) != 0 {
			t.Errorf("%s: seeds must not pin models", seed.Name)
		}
	}
	if !slices.Equal(names, []string{"Category", "Comparison", "Guide", "Hub", "Product"}) {
		t.Errorf("names = %v", names)
	}
}

func TestSeedRecipesUseTheStepCatalogue(t *testing.T) {
	t.Parallel()

	catalog := []string{"resolve_context", "generate_body", "generate_meta", "insert_links", "repair_links", "generate_images", "validate", "judge", "publish", "relink_neighbors", "sync_back", "report"}
	seeds := template.Seed()
	for i := range seeds {
		steps := make([]string, 0, len(seeds[i].Spec.Recipe))
		for _, step := range seeds[i].Spec.Recipe {
			steps = append(steps, step.Name)
		}
		if !slices.Equal(steps, catalog) {
			t.Errorf("%s recipe = %v, want the full catalog in order", seeds[i].Name, steps)
		}
	}
}

func TestSeedReturnsFreshCopies(t *testing.T) {
	t.Parallel()

	first := template.Seed()
	first[0].Spec.Sections[0].Heading = "mutated"
	if template.Seed()[0].Spec.Sections[0].Heading == "mutated" {
		t.Error("Seed must decode fresh values on every call")
	}
}
