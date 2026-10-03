package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
)

func TestTheAgentMakesAnEntityAWordPressCategoryAndBack(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	steps := []struct {
		name      string
		arguments string
		want      []string
	}{
		{
			name:      "filed",
			arguments: `{"id":"` + seeded.entity + `","siteCategory":true}`,
			want:      []string{`"siteCategory":true`, `"categories":[{"entityId":"` + seeded.entity + `","name":"Coffee"}]`},
		},
		{
			name:      "a new intent keeps it filed",
			arguments: `{"id":"` + seeded.entity + `","intent":"Find a coffee"}`,
			want:      []string{`"siteCategory":true`, `"intent":"Find a coffee"`, `"categories":[{"entityId":"` + seeded.entity + `","name":"Coffee"}]`},
		},
		{
			name:      "taken off",
			arguments: `{"id":"` + seeded.entity + `","siteCategory":false}`,
			want:      []string{`"siteCategory":false`, `"categories":[]`},
		},
	}

	for _, step := range steps {
		out, err := registry.Call(t.Context(), binding, "graph_update_entity", json.RawMessage(step.arguments))
		if err != nil {
			t.Fatalf("%s: graph_update_entity: %v", step.name, err)
		}
		encoded, err := json.Marshal(out)
		if err != nil {
			t.Fatalf("%s: encode the answer: %v", step.name, err)
		}
		for _, want := range step.want {
			if !strings.Contains(string(encoded), want) {
				t.Errorf("%s: the answer lacks %s: %s", step.name, want, encoded)
			}
		}
	}
}
