package steps_test

import (
	"testing"

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
		output   int
		calls    int
		unpriced bool
	}{
		{name: "generate_body", output: 0, calls: 1},
		{name: "generate_meta", output: 512, calls: 1},
		{name: "repair_links", output: 256, calls: 4},
		{name: "repair_links", params: map[string]any{"iterations": float64(1)}, output: 256, calls: 2},
		{name: "judge", output: 1024, calls: 1},
		{name: "generate_images", calls: 1, unpriced: true},
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

			calls := 1
			if def.Price.Calls != nil {
				calls = def.Price.Calls(spec, tc.params)
			}
			if calls != tc.calls {
				t.Errorf("calls = %d, want %d", calls, tc.calls)
			}
		})
	}
}
