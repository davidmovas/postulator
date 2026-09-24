package steps_test

import (
	"testing"

	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/runtime/steps"
)

func TestEveryStepSaysWhatOneItemOfItCosts(t *testing.T) {
	t.Parallel()

	registry := run.NewRegistry()
	if err := steps.Register(registry, steps.Deps{}); err != nil {
		t.Fatalf("Register: %v", err)
	}

	spec := template.TemplateSpec{LinkRules: template.LinkRules{UpDepth: 2}}
	cases := []struct {
		name     string
		params   map[string]any
		images   template.Images
		output   int
		calls    int
		lead     bool
		unpriced bool
	}{
		{name: "generate_body", output: 0, calls: 1},
		{name: "generate_meta", output: 512, calls: 1},
		{name: "repair_links", output: 256, calls: 4},
		{name: "repair_links", params: map[string]any{"iterations": float64(1)}, output: 256, calls: 2},
		{name: "repair_links", lead: true, output: 256, calls: 6},
		{name: "judge", output: 1024, calls: 1},
		{name: "generate_images", output: 1056, calls: 0, unpriced: true},
		{name: "generate_images", output: 1056, calls: 3, unpriced: true, images: template.Images{Featured: true, Inline: 2, Source: template.ImagesAI}},
		{name: "generate_images", output: 1056, calls: 0, unpriced: true, images: template.Images{Featured: true, Inline: 2, Source: template.ImagesWPMedia}},
		{name: "insert_links", calls: 1},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			def, known := registry.Lookup(tc.name)
			if !known {
				t.Fatalf("the registry does not hold %s", tc.name)
			}
			if def.Price.OutputTokens != tc.output {
				t.Errorf("outputTokens = %d, want %d", def.Price.OutputTokens, tc.output)
			}
			if def.Price.Unpriced != tc.unpriced {
				t.Errorf("unpriced = %t, want %t", def.Price.Unpriced, tc.unpriced)
			}

			priced := spec
			priced.KeywordRules.PrimaryInFirstParagraph = tc.lead
			priced.Images = tc.images
			calls := 1
			if def.Price.Calls != nil {
				calls = def.Price.Calls(priced, tc.params)
			}
			if calls != tc.calls {
				t.Errorf("calls = %d, want %d", calls, tc.calls)
			}
		})
	}
}

func TestTheImageStepIsPricedOnTheModelItIsGiven(t *testing.T) {
	t.Parallel()

	registry := run.NewRegistry()
	drawer := domainllm.ModelRef{Provider: "openai", Model: "gpt-image-2"}
	if err := steps.Register(registry, steps.Deps{ImageModel: &drawer}); err != nil {
		t.Fatalf("Register: %v", err)
	}
	def, known := registry.Lookup(steps.NameGenerateImages)
	if !known {
		t.Fatal("the registry does not hold generate_images")
	}
	if def.Price.Unpriced || def.Price.Ref == nil || *def.Price.Ref != drawer {
		t.Fatalf("price = %+v, want the image model priced", def.Price)
	}
}
