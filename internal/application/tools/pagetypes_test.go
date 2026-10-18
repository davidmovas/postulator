package tools_test

import (
	"encoding/json"
	"testing"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
)

func called[T any](t *testing.T, registry *tools.Registry, binding tools.Binding, name, args string) T {
	t.Helper()
	out, err := registry.Call(t.Context(), binding, name, json.RawMessage(args))
	if err != nil {
		t.Fatalf("%s %s: %v", name, args, err)
	}
	answer, ok := out.(T)
	if !ok {
		t.Fatalf("%s answered %T, want %T", name, out, answer)
	}
	return answer
}

func TestThePageTypeToolsSaveTypesScopesAndAPagesType(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	listed := called[pagetypes.ListResponse](t, registry, binding, "pagetypes_list", `{"siteId":"`+seeded.site+`"}`)
	var hub pagetypes.Type
	for i := range listed.Types {
		if listed.Types[i].Preset == "hub" {
			hub = listed.Types[i]
		}
	}
	if hub.ID == "" || len(listed.Fields) == 0 {
		t.Fatalf("pagetypes_list = %+v, want the seeded types and the field catalog", listed)
	}

	created := called[pagetypes.Type](t, registry, binding, "pagetypes_save",
		`{"siteId":"`+seeded.site+`","name":"Pillar","preset":"hub"}`)
	if created.Name != "Pillar" || created.Preset != "hub" {
		t.Fatalf("pagetypes_save without an id = %+v, want a new hub type", created)
	}
	renamed := called[pagetypes.Type](t, registry, binding, "pagetypes_save", `{"id":"`+created.ID+`","name":"Pillars"}`)
	if renamed.ID != created.ID || renamed.Name != "Pillars" || renamed.Preset != "hub" {
		t.Fatalf("pagetypes_save with an id = %+v, want the same type renamed", renamed)
	}

	scope := called[pagetypes.Scope](t, registry, binding, "pagetypes_set_scope",
		`{"pageTypeId":"`+hub.ID+`","field":"body","instruction":"Lead with the product grid"}`)
	if !scope.Enabled || scope.Instruction != "Lead with the product grid" || !scope.Links.Links || scope.Links.Rules.MaxLinks != 20 {
		t.Fatalf("pagetypes_set_scope = %+v, want the instruction set and everything else the scope had", scope)
	}

	custom := called[pagetypes.Scope](t, registry, binding, "pagetypes_set_scope",
		`{"pageTypeId":"`+hub.ID+`","metaKey":"spec_sheet","enabled":true,"label":"Spec sheet","provider":"plugin"}`)
	if custom.Field != "meta:spec_sheet" || !custom.Enabled || custom.Label != "Spec sheet" || custom.Provider != "plugin" || custom.Links.Links {
		t.Fatalf("pagetypes_set_scope with a meta key = %+v, want a new custom field scope without links", custom)
	}

	typed := called[pages.UpdateResponse](t, registry, binding, "pages_update",
		`{"id":"`+seeded.page+`","pageTypeId":"`+created.ID+`"}`)
	if typed.Page.PageTypeID == nil || *typed.Page.PageTypeID != created.ID || typed.Page.TypeSource != "manual" || typed.Page.TypeReason != "ai" {
		t.Fatalf("pages_update with a type = %+v, want the type recorded as the agent's", typed.Page)
	}
	titled := called[pages.UpdateResponse](t, registry, binding, "pages_update",
		`{"id":"`+seeded.page+`","title":"Coffee","pageTypeId":"`+hub.ID+`"}`)
	if titled.Page.Title != "Coffee" || titled.Page.PageTypeID == nil || *titled.Page.PageTypeID != hub.ID {
		t.Fatalf("pages_update with a title and a type = %+v, want both", titled.Page)
	}

	moved := called[pagetypes.DeleteResponse](t, registry, binding, "pagetypes_delete",
		`{"id":"`+created.ID+`","reassignTo":"`+hub.ID+`"}`)
	if moved.Moved != 0 {
		t.Fatalf("pagetypes_delete = %+v, want no page moved, the page having left the type", moved)
	}
}
