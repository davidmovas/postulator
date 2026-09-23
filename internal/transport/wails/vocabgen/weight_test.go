package main

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/application/tools"
)

const maxToolBytes = 120_000

func TestTheToolsATurnCarriesFitTheirBudget(t *testing.T) {
	t.Parallel()

	built := tools.New(tools.Deps{}).Build(tools.Binding{})
	total := 0
	widest := ""
	widestBytes := 0

	for i := range built {
		encoded, err := json.Marshal(map[string]any{
			"name":        built[i].Def.Name,
			"description": built[i].Def.Description,
			"parameters":  built[i].Def.Schema,
		})
		if err != nil {
			t.Fatalf("encode %s: %v", built[i].Def.Name, err)
		}
		total += len(encoded)
		if len(encoded) > widestBytes {
			widest, widestBytes = built[i].Def.Name, len(encoded)
		}
	}

	t.Logf("%d tools weigh %d bytes, about %d tokens; the widest is %s at %d bytes",
		len(built), total, total/4, widest, widestBytes)

	if total > maxToolBytes {
		t.Errorf("the tools weigh %d bytes, over the %d a turn may resend on every round",
			total, maxToolBytes)
	}
}
