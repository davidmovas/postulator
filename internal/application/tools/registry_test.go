package tools_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/llm"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func registered() []string {
	return []string{
		"sites_list",
		"sites_get",
		"sites_create",
		"sites_update",
		"sites_delete",
		"graph_list_entities",
		"graph_get_entity",
		"graph_create_entity",
		"graph_create_entities",
		"graph_update_entity",
		"graph_delete_entity",
		"graph_set_anchors",
		"graph_list_edges",
		"graph_add_edge",
		"graph_approve_edge",
		"graph_reject_edge",
		"graph_delete_edge",
		"graph_load",
		"graph_recompute_scores",
		"graph_propose_from_pages",
		"graph_propose_related",
		"pages_list",
		"pages_get",
		"pages_preview_link",
		"pages_create",
		"pages_update",
		"pages_delete",
		"pages_map_to_entity",
		"pages_unmap",
		"pages_set_canonical",
		"pages_tree",
		"pages_replace_links",
		"templates_list",
		"templates_get",
		"templates_create",
		"templates_update",
		"templates_delete",
		"templates_set_override",
		"templates_delete_override",
		"templates_resolve_for_page",
		"policies_list",
		"policies_get",
		"policies_create",
		"policies_update",
		"policies_delete",
		"policies_effective",
		"runs_start",
		"runs_get",
		"runs_list",
		"runs_list_items",
		"runs_list_events",
		"runs_get_artifact",
		"runs_pause",
		"runs_resume",
		"runs_cancel",
		"runs_retry_step",
		"sync_site",
		"sync_check_plugin",
		"reports_site_overview",
		"reports_page",
		"reports_run",
		"reports_link_audit",
		"reports_link_audit_page",
		"imports_inspect",
		"imports_preview",
		"imports_apply",
		"imports_export",
		"imports_save_mapping",
		"imports_list_mappings",
		"imports_delete_mapping",
		"models_list",
		"models_upsert",
		"models_disable",
		"models_set_profile",
		"models_get_profiles",
		"models_set_provider_key",
		"models_test_provider",
		"models_usage_summary",
		"content_judge_page",
		"schedules_create",
		"schedules_update",
		"schedules_delete",
		"schedules_get",
		"schedules_list",
		"schedules_enable",
		"schedules_disable",
		"schedules_run_now",
	}
}

type actionRecorder struct {
	actions []agent.PendingAction
	err     error
}

func (a *actionRecorder) Insert(_ context.Context, action agent.PendingAction) error {
	if a.err != nil {
		return a.err
	}
	a.actions = append(a.actions, action)
	return nil
}

type busRecorder struct {
	types    []events.Type
	payloads []any
}

func (b *busRecorder) Publish(eventType events.Type, payload any) error {
	b.types = append(b.types, eventType)
	b.payloads = append(b.payloads, payload)
	return nil
}

func newRegistry(actions *actionRecorder, bus *busRecorder) *tools.Registry {
	return tools.New(tools.Deps{Actions: actions, Publisher: bus, Clock: clock.NewFake(stamp)})
}

func TestEveryUseCaseIsRegisteredExactlyOnce(t *testing.T) {
	t.Parallel()

	names := newRegistry(&actionRecorder{}, &busRecorder{}).Names()
	want := registered()

	if !slices.Equal(names, want) {
		t.Fatalf("the registry offers %d tools and the hand kept list %d\nregistry: %v\nlist: %v",
			len(names), len(want), names, want)
	}

	seen := make(map[string]struct{}, len(names))
	for _, name := range names {
		if _, twice := seen[name]; twice {
			t.Errorf("%s is registered twice", name)
		}
		seen[name] = struct{}{}
	}
}

func TestAPreviewLinkIsApprovedBeforeItIsIssued(t *testing.T) {
	t.Parallel()

	registry := newRegistry(&actionRecorder{}, &busRecorder{})
	for _, tool := range registry.Build(tools.Binding{Mode: agent.ModeAutonomous}) {
		if tool.Def.Name != "pages_preview_link" {
			continue
		}
		if tool.Def.Risk != tools.RiskWrite {
			t.Fatalf("pages_preview_link risk = %s, want write: it hands a draft to anyone holding the link", tool.Def.Risk)
		}
		if tool.Def.Schema.Properties["pageId"] == nil {
			t.Fatalf("pages_preview_link schema = %+v, want a pageId", tool.Def.Schema)
		}
		return
	}
	t.Fatal("pages_preview_link is not registered")
}

func TestEveryToolCarriesADefinitionAndASchema(t *testing.T) {
	t.Parallel()

	registry := newRegistry(&actionRecorder{}, &busRecorder{})
	for _, tool := range registry.Build(tools.Binding{Mode: agent.ModeAutonomous}) {
		if tool.Def.Name == "" || tool.Def.Description == "" || !tool.Def.Risk.Valid() {
			t.Fatalf("tool definition = %+v", tool.Def)
		}
		if tool.Run == nil {
			t.Fatalf("%s has nothing to run", tool.Def.Name)
		}
		if tool.Def.Schema == nil || tool.Def.Schema.Type != llm.SchemaObject {
			t.Fatalf("%s carries the schema %+v", tool.Def.Name, tool.Def.Schema)
		}
		if tool.Authorize != nil && tool.Def.Schema.Properties["siteId"] != nil {
			t.Errorf("%s takes its site from the conversation, so the model must not be offered one", tool.Def.Name)
		}
	}
}

func TestSchemaIsDerivedFromTheRequest(t *testing.T) {
	t.Parallel()

	registry := newRegistry(&actionRecorder{}, &busRecorder{})
	def, known := registry.Lookup("pages_create")
	if !known {
		t.Fatal("pages_create is not registered")
	}

	if def.Schema.Properties["path"] == nil || def.Schema.Properties["path"].Type != llm.SchemaString {
		t.Fatalf("the path property is %+v", def.Schema.Properties["path"])
	}
	if def.Schema.Properties["entityId"] == nil {
		t.Fatal("an optional pointer field is still offered to the model")
	}
	if slices.Contains(def.Schema.Required, "entityId") {
		t.Error("an optional pointer field must not be required")
	}
	if slices.Contains(def.Schema.Required, "siteId") {
		t.Error("the bound site must not be required of the model")
	}

	edge, known := registry.Lookup("graph_add_edge")
	if !known {
		t.Fatal("graph_add_edge is not registered")
	}
	if edge.Schema.Properties["reason"] == nil || edge.Schema.Properties["reason"].Type != llm.SchemaString {
		t.Fatalf("the reason property is %+v", edge.Schema.Properties["reason"])
	}
	if slices.Contains(edge.Schema.Required, "reason") {
		t.Error("a reason is offered, never required")
	}
	if _, unknown := registry.Lookup("no_such_tool"); unknown {
		t.Error("Lookup answered for a tool that is not registered")
	}
}

func TestASiteScopedToolRefusesAConversationWithoutASite(t *testing.T) {
	t.Parallel()

	registry := newRegistry(&actionRecorder{}, &busRecorder{})
	_, err := registry.Call(t.Context(), tools.Binding{Mode: agent.ModeAutonomous}, "pages_tree", nil)
	if !errors.IsCode(err, errors.Unauthorized) {
		t.Fatalf("Call without a site = %v, want unauthorized", err)
	}

	_, err = registry.Call(t.Context(), tools.Binding{Mode: agent.ModeAutonomous}, "no_such_tool", nil)
	if !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Call of an unknown tool = %v, want not found", err)
	}
}

func TestConfirmModeProposesInsteadOfWriting(t *testing.T) {
	t.Parallel()

	actions := &actionRecorder{}
	bus := &busRecorder{}
	registry := newRegistry(actions, bus)
	binding := tools.Binding{SiteID: "site-1", ConversationID: "conversation-1", Mode: agent.ModeConfirm}

	var proposed, read tools.Tool
	for _, tool := range registry.Build(binding) {
		switch tool.Def.Name {
		case "pages_delete":
			proposed = tool
		case "pages_tree":
			read = tool
		}
	}

	out, err := proposed.Run(t.Context(), binding, json.RawMessage(`{"id":"page-1"}`))
	if err != nil {
		t.Fatalf("a proposed call = %v", err)
	}

	confirmation, ok := out.(tools.Confirmation)
	if !ok || confirmation.Status != agent.ConfirmationRequired || confirmation.ActionID == "" {
		t.Fatalf("a proposed call returned %+v", out)
	}
	if len(actions.actions) != 1 || actions.actions[0].Tool != "pages_delete" {
		t.Fatalf("the stored actions are %+v", actions.actions)
	}
	if actions.actions[0].Summary != confirmation.Summary || confirmation.Summary == "" {
		t.Fatalf("the summary is %q", confirmation.Summary)
	}
	if len(bus.types) != 1 || bus.types[0] != events.AgentConfirmRequested {
		t.Fatalf("the bus saw %v", bus.types)
	}

	payload, ok := bus.payloads[0].(events.AgentConfirmRequestedPayload)
	if !ok || payload.ConversationID != "conversation-1" || payload.Risk != string(tools.RiskDangerous) {
		t.Fatalf("the payload is %+v", bus.payloads[0])
	}

	if read.Authorize == nil {
		t.Fatal("pages_tree is site scoped and must authorize")
	}
	if len(actions.actions) != 1 {
		t.Fatal("a read tool must not propose anything")
	}
}

func TestAProposalNeedsItsConversation(t *testing.T) {
	t.Parallel()

	registry := newRegistry(&actionRecorder{}, &busRecorder{})
	binding := tools.Binding{SiteID: "site-1", Mode: agent.ModeConfirm}

	for _, tool := range registry.Build(binding) {
		if tool.Def.Name != "pages_delete" {
			continue
		}
		if _, err := tool.Run(t.Context(), binding, nil); !errors.IsCode(err, errors.Invalid) {
			t.Fatalf("a proposal without a conversation = %v", err)
		}
	}
}

func TestRiskDecidesWhatNeedsConfirmation(t *testing.T) {
	t.Parallel()

	if !tools.RiskWrite.NeedsConfirmation() || !tools.RiskDangerous.NeedsConfirmation() {
		t.Error("a write and a dangerous tool are confirmed")
	}
	if tools.RiskRead.NeedsConfirmation() {
		t.Error("a read tool is never confirmed")
	}
	if tools.Risk("spicy").Valid() {
		t.Error("an unknown risk must not validate")
	}
}

func TestSummaryCarriesTheArguments(t *testing.T) {
	t.Parallel()

	def := tools.Def{Name: "pages_delete", Description: "Delete a page from the page map."}
	if got := tools.Summary(def, json.RawMessage(`{}`)); got != "Delete a page from the page map" {
		t.Fatalf("Summary of no arguments = %q", got)
	}
	if got := tools.Summary(def, json.RawMessage(`{"id":"p1"}`)); got != `Delete a page from the page map with {"id":"p1"}` {
		t.Fatalf("Summary = %q", got)
	}

	long := make([]byte, 0, 400)
	long = append(long, []byte(`{"id":"`)...)
	for range 300 {
		long = append(long, 'a')
	}
	long = append(long, []byte(`"}`)...)
	if got := tools.Summary(def, long); len(got) >= len(long) {
		t.Fatalf("Summary of a long argument list was not cut: %d", len(got))
	}
	if got := tools.Summary(tools.Def{Name: "bare"}, nil); got != "bare" {
		t.Fatalf("Summary without a description = %q", got)
	}
}

func TestAnUnusableCallNeverBecomesAConfirmation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		tool string
		args string
	}{
		{
			name: "a field the tool does not take",
			tool: "graph_create_entity",
			args: `{"name":"Creatine","kind":"topic","shade":"blue"}`,
		},
		{
			name: "a field of the wrong shape",
			tool: "templates_create",
			args: `{"name":"Guide","pageKind":"guide","spec":{"tone":{"voice":"warm"}}}`,
		},
		{
			name: "a specification the domain refuses",
			tool: "templates_create",
			args: `{"name":"Guide","pageKind":"guide","spec":{"sections":[],"tone":"plain"}}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			actions := &actionRecorder{}
			bus := &busRecorder{}
			registry := newRegistry(actions, bus)
			binding := tools.Binding{ConversationID: "conversation-1", SiteID: "site-1", Mode: agent.ModeConfirm}

			_, err := registry.Call(t.Context(), binding, tc.tool, json.RawMessage(tc.args))
			if !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("the call answered %v, want an invalid argument", err)
			}
			if len(actions.actions) != 0 {
				t.Fatalf("the client was asked to approve %+v", actions.actions)
			}
			if len(bus.payloads) != 0 {
				t.Fatalf("a confirmation was announced: %+v", bus.payloads)
			}
		})
	}
}

func TestASecretNeverReachesTheSummaryOrTheEvent(t *testing.T) {
	t.Parallel()

	actions := &actionRecorder{}
	bus := &busRecorder{}
	registry := newRegistry(actions, bus)
	binding := tools.Binding{ConversationID: "conversation-1", Mode: agent.ModeConfirm}

	args := json.RawMessage(`{"provider":"openai","apiKey":"sk-live-secret"}`)
	for _, tool := range registry.Build(binding) {
		if tool.Def.Name != "models_set_provider_key" {
			continue
		}
		if _, err := tool.Run(t.Context(), binding, args); err != nil {
			t.Fatalf("a proposed call = %v", err)
		}
	}

	if len(actions.actions) != 1 {
		t.Fatalf("the stored actions are %+v", actions.actions)
	}
	if strings.Contains(actions.actions[0].Summary, "sk-live-secret") {
		t.Fatalf("the summary carries the key: %q", actions.actions[0].Summary)
	}
	if !strings.Contains(string(actions.actions[0].Args), "sk-live-secret") {
		t.Fatal("the stored action must keep the arguments it will replay")
	}

	payload, ok := bus.payloads[0].(events.AgentConfirmRequestedPayload)
	if !ok || strings.Contains(string(payload.Args), "sk-live-secret") {
		t.Fatalf("the event carries the key: %s", payload.Args)
	}
	nested := tools.Redact(json.RawMessage(`{"rows":[{"password":"hunter2"}]}`))
	if strings.Contains(string(nested), "hunter2") {
		t.Fatalf("a nested secret survived the redaction: %s", nested)
	}

	if got := string(tools.Redact(json.RawMessage("not json"))); got != "not json" {
		t.Fatalf("Redact of something unreadable = %s", got)
	}
	if got := string(tools.Redact(nil)); got != "" {
		t.Fatalf("Redact of nothing = %q", got)
	}
}
