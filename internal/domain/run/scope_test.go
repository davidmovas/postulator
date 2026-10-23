package run_test

import (
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/field"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

func enabledNames(recipe []template.StepSpec) []string {
	out := make([]string, 0, len(recipe))
	for _, spec := range recipe {
		if spec.Enabled {
			out = append(out, spec.Name)
		}
	}
	return out
}

func TestARunScopedToFieldsKeepsOnlyTheStepsTheyNeed(t *testing.T) {
	t.Parallel()

	always := []string{"publish", "sync_back", "report"}
	body := []string{
		"resolve_context", "generate_body", "insert_links", "repair_links", "generate_images", "validate", "judge",
		"publish", "relink_neighbors", "sync_back", "report",
	}
	cases := []struct {
		name   string
		fields []string
		want   []string
	}{
		{name: "the meta description", fields: []string{"seo_description"}, want: append([]string{"generate_meta"}, always...)},
		{name: "both seo fields", fields: []string{"seo_title", "seo_description"}, want: append([]string{"generate_meta"}, always...)},
		{name: "the excerpt", fields: []string{"excerpt"}, want: append([]string{"generate_fields"}, always...)},
		{name: "the title without the body", fields: []string{"title"}, want: append([]string{"generate_fields"}, always...)},
		{name: "a variation's description", fields: []string{"variation_description:12"}, want: append([]string{"generate_fields"}, always...)},
		{name: "a custom field", fields: []string{"meta:dosage"}, want: append([]string{"generate_fields"}, always...)},
		{name: "the upsells", fields: []string{"upsells"}, want: append([]string{"generate_fields"}, always...)},
		{name: "the body", fields: []string{"body"}, want: body},
		{name: "the title with the body", fields: []string{"title", "body"}, want: body},
		{name: "a product's description", fields: []string{"product_description"}, want: body},
		{
			name: "the featured image", fields: []string{"featured_image"},
			want: []string{"resolve_context", "generate_body", "generate_images", "publish", "sync_back", "report"},
		},
		{
			name: "the meta and the excerpt", fields: []string{"excerpt", "seo_title"},
			want: append([]string{"generate_meta", "generate_fields"}, always...),
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			scoped, err := run.ScopeRecipe(run.GenerateRecipe(), tc.fields)
			if err != nil {
				t.Fatalf("ScopeRecipe(%v): %v", tc.fields, err)
			}
			if got := enabledNames(scoped); !slices.Equal(got, tc.want) {
				t.Fatalf("ScopeRecipe(%v) keeps %v, want %v", tc.fields, got, tc.want)
			}
			listed := slices.Sorted(slices.Values(tc.fields))
			for _, spec := range scoped {
				sc := run.StepContext{Params: spec.Params}
				got := make([]string, 0, len(listed))
				for _, key := range sc.Fields() {
					got = append(got, string(key))
				}
				switch {
				case spec.Enabled && !slices.Equal(got, listed):
					t.Errorf("%s carries the scope %v, want %v", spec.Name, got, listed)
				case !spec.Enabled && len(got) > 0:
					t.Errorf("the disabled %s carries the scope %v, want none", spec.Name, got)
				}
			}
		})
	}
}

func TestAScopeKeepsAStepsOwnParamsAndListsEachFieldOnce(t *testing.T) {
	t.Parallel()

	recipe := run.GenerateRecipe()
	for i := range recipe {
		if recipe[i].Name == "publish" {
			recipe[i].Params = map[string]any{"refuseDrift": true}
		}
	}
	scoped, err := run.ScopeRecipe(recipe, []string{"seo_title", "seo_description", "seo_title"})
	if err != nil {
		t.Fatalf("ScopeRecipe: %v", err)
	}
	publish := run.ParamsFor(scoped, "publish")
	if publish["refuseDrift"] != true {
		t.Errorf("publish's params are %v, want refuseDrift kept", publish)
	}
	sc := run.StepContext{Params: publish}
	if got := sc.Fields(); !slices.Equal(got, []field.Key{field.SEODescription, field.SEOTitle}) {
		t.Errorf("publish is scoped to %v, want each field once, sorted", got)
	}
	if recipe[0].Params != nil || !recipe[1].Enabled {
		t.Error("ScopeRecipe changed the recipe it was given")
	}
	unscoped, err := run.ScopeRecipe(recipe, nil)
	if err != nil || !slices.Equal(enabledNames(unscoped), enabledNames(recipe)) {
		t.Errorf("ScopeRecipe(nil) = %v, %v, want the recipe unchanged", enabledNames(unscoped), err)
	}
}

func TestAScopeIsRefusedWhenTheRunMayNotWriteIt(t *testing.T) {
	t.Parallel()

	withoutMeta := run.GenerateRecipe()
	withoutPublish := run.GenerateRecipe()
	for i := range withoutMeta {
		withoutMeta[i].Enabled = withoutMeta[i].Name != "generate_meta"
		withoutPublish[i].Enabled = withoutPublish[i].Name != "publish"
	}
	cases := []struct {
		name   string
		recipe []template.StepSpec
		fields []string
	}{
		{name: "a denied field", recipe: run.GenerateRecipe(), fields: []string{"sku"}},
		{name: "a read-only field", recipe: run.GenerateRecipe(), fields: []string{"h1"}},
		{name: "a person's switch", recipe: run.GenerateRecipe(), fields: []string{"seo_canonical"}},
		{name: "an unknown field", recipe: run.GenerateRecipe(), fields: []string{"tagline"}},
		{name: "an instance of a plain field", recipe: run.GenerateRecipe(), fields: []string{"excerpt:2"}},
		{name: "no publish", recipe: withoutPublish, fields: []string{"excerpt"}},
		{name: "no step writes it", recipe: withoutMeta, fields: []string{"seo_description"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := run.ScopeRecipe(tc.recipe, tc.fields); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("ScopeRecipe(%v) = %v, want invalid", tc.fields, err)
			}
		})
	}
}

func TestAScopedPublishRequiresOnlyTheDraftsOfItsFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		fields []field.Key
		want   []run.ArtifactKind
	}{
		{name: "the meta", fields: []field.Key{field.SEODescription}, want: []run.ArtifactKind{run.ArtifactMeta}},
		{name: "the excerpt", fields: []field.Key{field.Excerpt}, want: []run.ArtifactKind{run.ArtifactFields}},
		{name: "the title alone", fields: []field.Key{field.Title}, want: []run.ArtifactKind{run.ArtifactFields}},
		{
			name: "the title with the body", fields: []field.Key{field.Body, field.Title},
			want: []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
		},
		{name: "the featured image", fields: []field.Key{field.FeaturedImage}, want: []run.ArtifactKind{run.ArtifactImages}},
		{
			name: "the meta and a variation", fields: []field.Key{field.SEOTitle, field.Instance(field.VariationDescription, "4")},
			want: []run.ArtifactKind{run.ArtifactMeta, run.ArtifactFields},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := run.DraftsOf(tc.fields); !slices.Equal(got, tc.want) {
				t.Errorf("DraftsOf(%v) = %v, want %v", tc.fields, got, tc.want)
			}
		})
	}
}

func TestAStepsRequirementsFollowTheScope(t *testing.T) {
	t.Parallel()

	registry := run.NewRegistry()
	defs := []run.StepDef{
		{Name: "resolve_context", Produces: []run.ArtifactKind{run.ArtifactLinkContext}, Run: noop},
		{
			Name: "generate_body", Requires: []run.ArtifactKind{run.ArtifactLinkContext},
			Produces: []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML}, Run: noop,
		},
		{Name: "generate_meta", Produces: []run.ArtifactKind{run.ArtifactMeta}, Run: noop},
		{
			Name: "publish", Requires: []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML},
			Scoped: run.DraftsOf, Produces: []run.ArtifactKind{run.ArtifactPublishResult}, Run: noop,
		},
	}
	for i := range defs {
		if err := registry.Register(defs[i]); err != nil {
			t.Fatalf("Register(%s): %v", defs[i].Name, err)
		}
	}
	recipe := []template.StepSpec{
		{Name: "resolve_context", Enabled: true}, {Name: "generate_body", Enabled: true},
		{Name: "generate_meta", Enabled: true}, {Name: "publish", Enabled: true},
	}

	scoped, err := run.ScopeRecipe(recipe, []string{"seo_description"})
	if err != nil {
		t.Fatalf("ScopeRecipe: %v", err)
	}
	if err := run.ValidateRecipe(registry, scoped); err != nil {
		t.Errorf("a recipe scoped to the meta without the body: %v, want it valid", err)
	}
	unscoped := slices.Clone(recipe)
	unscoped[1].Enabled = false
	if err := run.ValidateRecipe(registry, unscoped); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unscoped publish without the body = %v, want invalid", err)
	}

	_, publish, _ := mustPlan(t, registry, scoped).StepByName("publish")
	if got := publish.RequiresFor(run.ParamsFor(scoped, "publish")); !slices.Equal(got, []run.ArtifactKind{run.ArtifactMeta}) {
		t.Errorf("the scoped publish requires %v, want the meta alone", got)
	}
	if got := publish.RequiresFor(nil); !slices.Equal(got, []run.ArtifactKind{run.ArtifactDraft, run.ArtifactBodyHTML}) {
		t.Errorf("the unscoped publish requires %v, want the draft and the body", got)
	}
}

func mustPlan(t *testing.T, registry *run.Registry, recipe []template.StepSpec) run.Definition {
	t.Helper()

	def, err := run.Plan(registry, run.KindGenerate, recipe, time.Hour)
	if err != nil {
		t.Fatalf("Plan: %v", err)
	}
	return def
}
