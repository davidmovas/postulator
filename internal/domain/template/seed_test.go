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

func TestTheBuiltInsAskForNoImage(t *testing.T) {
	t.Parallel()

	seeds := template.Seed()
	for i := range seeds {
		if wanted := seeds[i].Spec.Images.Wanted(); wanted != 0 {
			t.Errorf("%s asks for %d images, want a built-in without images", seeds[i].Name, wanted)
		}
	}
}

func TestEverySupersededSpecPrecedesASeed(t *testing.T) {
	t.Parallel()

	seeds := template.Seed()
	superseded := template.Superseded()
	if len(superseded) == 0 {
		t.Fatal("no superseded built-in is shipped")
	}
	for i := range superseded {
		old := superseded[i]
		index := slices.IndexFunc(seeds, func(seed template.Template) bool { return seed.Name == old.Name })
		if index < 0 {
			t.Errorf("the superseded %s names no current built-in", old.Name)
			continue
		}
		current := seeds[index]
		if old.PageKind != current.PageKind || old.Scope != template.ScopeGlobal {
			t.Errorf("the superseded %s = %+v, want the kind and scope of the current one", old.Name, old)
		}
		if template.Supersedes(current, current) {
			t.Errorf("the current %s supersedes itself, so every start would refresh it again", current.Name)
		}
		if !template.Supersedes(current, old) {
			t.Errorf("the shipped %s does not refresh its earlier copy", current.Name)
		}
		if err := template.Validate(old.Spec); err != nil {
			t.Errorf("the superseded %s: %v", old.Name, err)
		}
	}
}

func TestSupersedes(t *testing.T) {
	t.Parallel()

	seeds := template.Seed()
	hub := seeds[slices.IndexFunc(seeds, func(seed template.Template) bool { return seed.Name == "Hub" })]
	superseded := template.Superseded()
	old := superseded[slices.IndexFunc(superseded, func(seed template.Template) bool { return seed.Name == "Hub" })]
	site := "site-1"

	cases := []struct {
		name   string
		stored func() template.Template
		want   bool
	}{
		{name: "an untouched earlier copy", stored: func() template.Template { return old }, want: true},
		{name: "the current copy", stored: func() template.Template { return hub }},
		{
			name: "an edited earlier copy",
			stored: func() template.Template {
				edited := template.Superseded()[slices.IndexFunc(superseded, func(seed template.Template) bool { return seed.Name == "Hub" })]
				edited.Spec.Tone = "Warm"
				return edited
			},
		},
		{
			name: "another page kind",
			stored: func() template.Template {
				other := old
				other.PageKind = "guide"
				return other
			},
		},
		{
			name: "a site template of the same name",
			stored: func() template.Template {
				scoped := old
				scoped.Scope = template.ScopeSite
				scoped.SiteID = &site
				return scoped
			},
		},
		{
			name: "the name in another case",
			stored: func() template.Template {
				lower := old
				lower.Name = "hub"
				return lower
			},
			want: true,
		},
		{
			name: "another name",
			stored: func() template.Template {
				renamed := old
				renamed.Name = "Hub copy"
				return renamed
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := template.Supersedes(hub, tc.stored()); got != tc.want {
				t.Fatalf("Supersedes = %v, want %v", got, tc.want)
			}
		})
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
