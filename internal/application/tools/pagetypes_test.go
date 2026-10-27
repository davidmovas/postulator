package tools_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/pagetypes"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
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

func TestTheAgentCannotLetARunRewriteTheOwnersText(t *testing.T) {
	t.Parallel()

	registry, binding, _ := wired(t)
	def, found := registry.Lookup("pagetypes")
	if !found {
		t.Fatal("pagetypes is not registered")
	}
	if _, offered := def.Schema.Properties["rewrites"]; offered {
		t.Error("pagetypes offers rewrites, which only a person may switch on, in the page type's screen or the run start")
	}
	binding.Mode = domainagent.ModeAutonomous
	listed := called[pagetypes.ListResponse](t, registry, binding, "pagetypes", `{"op":"list"}`)
	for i := range listed.Types {
		typeID := listed.Types[i].ID
		_, err := registry.Call(t.Context(), binding, "pagetypes",
			json.RawMessage(`{"op":"scope","pageTypeId":"`+typeID+`","field":"body","rewrites":true}`))
		after := called[pagetypes.ListResponse](t, registry, binding, "pagetypes", `{"op":"list"}`)
		for j := range after.Types {
			for _, scope := range after.Types[j].Scopes {
				if scope.Rewrites {
					t.Fatalf("after a call naming rewrites (%v) the %s scope of %s rewrites", err, scope.Field, after.Types[j].Name)
				}
			}
		}
	}
}

func TestThePageTypeToolsSaveTypesScopesAndAPagesType(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous

	listed := called[pagetypes.ListResponse](t, registry, binding, "pagetypes", `{"op":"list"}`)
	var hub pagetypes.Type
	for i := range listed.Types {
		if listed.Types[i].Preset == "category" {
			hub = listed.Types[i]
		}
	}
	if hub.ID == "" || len(listed.Fields) == 0 {
		t.Fatalf("pagetypes list = %+v, want the seeded types and the field catalog", listed)
	}

	created := called[pagetypes.Type](t, registry, binding, "pagetypes", `{"op":"save","name":"Pillar","preset":"hub"}`)
	if created.Name != "Pillar" || created.Preset != "hub" || created.SiteID != seeded.site {
		t.Fatalf("pagetypes save without an id = %+v, want a new hub type of the conversation's site", created)
	}
	renamed := called[pagetypes.Type](t, registry, binding, "pagetypes", `{"op":"save","pageTypeId":"`+created.ID+`","name":"Pillars"}`)
	if renamed.ID != created.ID || renamed.Name != "Pillars" || renamed.Preset != "hub" {
		t.Fatalf("pagetypes save with an id = %+v, want the same type renamed", renamed)
	}

	scope := called[pagetypes.Scope](t, registry, binding, "pagetypes",
		`{"op":"scope","pageTypeId":"`+hub.ID+`","field":"body","instruction":"Lead with the product grid"}`)
	if !scope.Enabled || scope.Instruction != "Lead with the product grid" || !scope.Links.Links || scope.Links.Rules.MaxLinks != 20 {
		t.Fatalf("pagetypes scope = %+v, want the instruction set and everything else the scope had", scope)
	}

	custom := called[pagetypes.Scope](t, registry, binding, "pagetypes",
		`{"op":"scope","pageTypeId":"`+hub.ID+`","metaKey":"spec_sheet","enabled":true,"label":"Spec sheet","provider":"plugin"}`)
	if custom.Field != "meta:spec_sheet" || !custom.Enabled || custom.Label != "Spec sheet" || custom.Provider != "plugin" || custom.Links.Links {
		t.Fatalf("pagetypes scope with a meta key = %+v, want a new custom field scope without links", custom)
	}

	typed := called[pages.UpdateResponse](t, registry, binding, "pages_manage",
		`{"op":"update","pageId":"`+seeded.page+`","pageTypeId":"`+created.ID+`"}`)
	if typed.Page.PageTypeID == nil || *typed.Page.PageTypeID != created.ID || typed.Page.TypeSource != "manual" || typed.Page.TypeReason != "ai" {
		t.Fatalf("pages_manage update with a type = %+v, want the type recorded as the agent's", typed.Page)
	}
	titled := called[pages.UpdateResponse](t, registry, binding, "pages_manage",
		`{"op":"update","pageId":"`+seeded.page+`","title":"Coffee","pageTypeId":"`+hub.ID+`"}`)
	if titled.Page.Title != "Coffee" || titled.Page.PageTypeID == nil || *titled.Page.PageTypeID != hub.ID {
		t.Fatalf("pages_manage update with a title and a type = %+v, want both", titled.Page)
	}
	moved := called[pagetypes.DeleteResponse](t, registry, binding, "pagetypes",
		`{"op":"delete","pageTypeId":"`+created.ID+`","reassignTo":"`+hub.ID+`"}`)
	if moved.Moved != 0 {
		t.Fatalf("pagetypes delete = %+v, want no page moved, the page having left the type", moved)
	}

	other := called[pagetypes.Type](t, registry, binding, "pagetypes", `{"op":"save","name":"Second hub","preset":"hub"}`)
	bulk := called[pages.SetTypesResponse](t, registry, binding, "pages_manage",
		`{"op":"set_types","pageTypeId":"`+other.ID+`","select":{"pageIds":["`+seeded.page+`"]}}`)
	if bulk.Typed != 1 || typeOfThePage(t, seeded) != other.ID {
		t.Fatalf("pages_manage set_types = %+v, want the one page typed", bulk)
	}
}

func TestTheRulesAreReadChangedOrderedAndTestedThroughTheClassification(t *testing.T) {
	t.Parallel()

	registry, binding, seeded := wired(t)
	binding.Mode = domainagent.ModeAutonomous
	onTheSite(t, seeded)
	called[pages.UpdateResponse](t, registry, binding, "pages_manage", `{"op":"update","pageId":"`+seeded.page+`","status":"published"}`)
	listed := called[pagetypes.ListResponse](t, registry, binding, "pagetypes", `{"op":"list"}`)
	hub := listed.Types[0]

	rules := called[tools.PagetypesClassifyResponse](t, registry, binding, "pagetypes_classify", `{"op":"rules"}`)
	if rules.Rules == nil || rules.Rules.Fallback == nil {
		t.Fatalf("pagetypes_classify rules = %+v, want the site's rules and its fallback", rules.Rules)
	}
	saved := called[tools.PagetypesClassifyResponse](t, registry, binding, "pagetypes_classify",
		`{"op":"save_rule","name":"Coffee pages","typeId":"`+hub.ID+`","conditions":{"urls":[{"mode":"prefix","value":"/coffee/"}]}}`)
	if saved.Rule == nil || saved.Rule.Name != "Coffee pages" || !saved.Rule.Enabled || saved.Rule.TypeID != hub.ID {
		t.Fatalf("pagetypes_classify save_rule = %+v, want the new rule on", saved.Rule)
	}
	tested := called[tools.PagetypesClassifyResponse](t, registry, binding, "pagetypes_classify",
		`{"op":"test_rule","ruleId":"`+saved.Rule.ID+`","pageId":"`+seeded.page+`"}`)
	if tested.Tested == nil || tested.Tested.Verdict != "met" {
		t.Errorf("pagetypes_classify test_rule = %+v, want /coffee/ to meet its prefix", tested.Tested)
	}
	after := called[tools.PagetypesClassifyResponse](t, registry, binding, "pagetypes_classify", `{"op":"rules"}`)
	order := make([]string, 0, len(after.Rules.Rules))
	for i := len(after.Rules.Rules) - 1; i >= 0; i-- {
		order = append(order, `"`+after.Rules.Rules[i].ID+`"`)
	}
	ordered := called[tools.PagetypesClassifyResponse](t, registry, binding, "pagetypes_classify",
		`{"op":"reorder_rules","ruleIds":[`+strings.Join(order, ",")+`]}`)
	if ordered.Rules == nil || ordered.Rules.Rules[0].ID != after.Rules.Rules[len(after.Rules.Rules)-1].ID {
		t.Errorf("pagetypes_classify reorder_rules = %+v, want the order reversed", ordered.Rules)
	}
	called[tools.PagetypesClassifyResponse](t, registry, binding, "pagetypes_classify", `{"op":"delete_rule","ruleId":"`+saved.Rule.ID+`"}`)
	if _, err := registry.Call(t.Context(), binding, "pagetypes_classify", json.RawMessage(`{"op":"weigh"}`)); !errors.IsCode(err, errors.Invalid) {
		t.Errorf("an unknown op = %v, want Invalid", err)
	}
}
