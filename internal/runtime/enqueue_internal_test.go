package runtime

import (
	"testing"

	"github.com/davidmovas/postulator/internal/domain/template"
)

func TestARunPublishesOnlyWhenItsRecipeEnablesThePublishStep(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		recipe []template.StepSpec
		want   bool
	}{
		{name: "an empty recipe", recipe: nil},
		{
			name:   "a recipe that writes and publishes",
			recipe: []template.StepSpec{{Name: "generate_body", Enabled: true}, {Name: "publish", Enabled: true}},
			want:   true,
		},
		{
			name:   "a recipe that turns publishing off",
			recipe: []template.StepSpec{{Name: "generate_body", Enabled: true}, {Name: "publish"}},
		},
		{
			name:   "a recipe without the publish step",
			recipe: []template.StepSpec{{Name: "resolve_context", Enabled: true}, {Name: "relink_page", Enabled: true}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := publishes(tc.recipe); got != tc.want {
				t.Fatalf("publishes = %t, want %t", got, tc.want)
			}
		})
	}
}
