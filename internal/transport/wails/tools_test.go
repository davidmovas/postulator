package wails_test

import (
	"context"
	"testing"

	"go.uber.org/zap"

	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

type catalogFake struct {
	names  []string
	defs   map[string]tools.Def
	panics bool
}

func (c catalogFake) Names() []string {
	if c.panics {
		panic("registry went away")
	}
	return c.names
}

func (c catalogFake) Lookup(name string) (tools.Def, bool) {
	def, ok := c.defs[name]
	return def, ok
}

func TestToolsServiceListsTheAgentCapabilities(t *testing.T) {
	t.Parallel()

	schema := &llm.Schema{Type: llm.SchemaObject, Required: []string{"pageId"}}
	catalog := catalogFake{
		names: []string{"pages_get", "pages_delete", "pages_gone"},
		defs: map[string]tools.Def{
			"pages_get":    {Name: "pages_get", Description: "read one page", Risk: tools.RiskRead, Schema: schema},
			"pages_delete": {Name: "pages_delete", Description: "remove one page", Risk: tools.RiskDangerous},
		},
	}

	listed, err := wails.NewToolsService(zap.NewNop(), ready[wails.ToolCatalog](catalog)).List(context.Background(), wails.ListToolsRequest{})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed.Tools) != 2 {
		t.Fatalf("List returned %d tools, want the two the registry can look up", len(listed.Tools))
	}

	first := listed.Tools[0]
	if first.Name != "pages_get" || first.Description != "read one page" || first.Risk != "read" {
		t.Errorf("first tool = %+v, want the registry order and its risk as text", first)
	}
	if first.Schema != schema {
		t.Errorf("first tool schema = %+v, want the declared schema", first.Schema)
	}
	if listed.Tools[1].Risk != "dangerous" {
		t.Errorf("second tool risk = %q, want %q", listed.Tools[1].Risk, "dangerous")
	}
}

func TestToolsServiceConvertsEveryFailure(t *testing.T) {
	t.Parallel()

	assertMethodNames(t, wails.NewToolsService(zap.NewNop(), ready[wails.ToolCatalog](catalogFake{})), []string{"List"})
	assertEveryMethodConverts(t, wails.NewToolsService(zap.NewNop(), ready[wails.ToolCatalog](catalogFake{panics: true})), panicBody)
}
