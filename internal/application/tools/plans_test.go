package tools_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/tools"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type planRecorder struct {
	plans []agent.Plan
}

func (p *planRecorder) Insert(_ context.Context, plan agent.Plan) error {
	p.plans = append(p.plans, plan)
	return nil
}

func (p *planRecorder) Get(_ context.Context, id string) (agent.Plan, error) {
	for i := range p.plans {
		if p.plans[i].ID == id {
			return p.plans[i], nil
		}
	}
	return agent.Plan{}, errors.New(errors.NotFound, "plan not found")
}

func (p *planRecorder) ListByConversation(_ context.Context, conversationID string) ([]agent.Plan, error) {
	out := make([]agent.Plan, 0)
	for i := range p.plans {
		if p.plans[i].ConversationID == conversationID {
			out = append(out, p.plans[i])
		}
	}
	return out, nil
}

func planCall(t *testing.T, registry *tools.Registry, b tools.Binding, args string) (map[string]any, error) {
	t.Helper()

	out, err := registry.Call(t.Context(), b, "plans", json.RawMessage(args))
	if err != nil {
		return nil, err
	}
	encoded, marshalErr := json.Marshal(out)
	if marshalErr != nil {
		t.Fatalf("encode the answer: %v", marshalErr)
	}
	var decoded map[string]any
	if unmarshalErr := json.Unmarshal(encoded, &decoded); unmarshalErr != nil {
		t.Fatalf("decode the answer: %v", unmarshalErr)
	}
	return decoded, nil
}

func TestAPlansViewOffersEachEditableFieldWithItsValue(t *testing.T) {
	t.Parallel()

	view := tools.ViewOfPlan(agent.Plan{ID: "p", Title: "t", Steps: []agent.PlanStep{{
		ID: "s1", Tool: "runs_control", Args: json.RawMessage(`{"op":"start","brief":{"goal":"Sell the kit"},"anchor":"kits"}`),
		Editable: []agent.EditField{{Name: "goal", Path: "brief.goal"}, {Name: "anchor", Path: "anchor"}, {Name: "tone", Path: "brief.tone"}},
	}}})
	edits := view.Steps[0].Edits
	if len(edits) != 3 || edits[0].Name != "goal" || edits[0].Value != "Sell the kit" || edits[1].Value != "kits" || edits[2].Value != "" {
		t.Fatalf("the editable fields = %+v", edits)
	}
}

func TestAPlanIsProposedOnlyOfWritesTheirToolsWouldTake(t *testing.T) {
	t.Parallel()

	plans, bus := &planRecorder{}, &busRecorder{}
	registry := tools.New(tools.Deps{Actions: &actionRecorder{}, Plans: plans, Publisher: bus, Clock: clock.NewFake(stamp)})
	b := tools.Binding{SiteID: "site-1", ConversationID: "conv-1", Mode: agent.ModeAutonomous}

	answer, err := planCall(t, registry, b, `{"op":"propose","title":"Tie the serums together","steps":[
		{"tool":"graph_add_edge","args":"{\"fromEntityId\":\"e-retinol\",\"toEntityId\":\"e-vitc\",\"kind\":\"related\"}","reason":"Both are serums the page tree keeps side by side"},
		{"tool":"graph_move_entity","args":"{\"entityId\":\"e-retinol\",\"newParentId\":\"e-serums\"}","reason":"The page tree puts it under Serums","dependsOn":[1]}
	]}`)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if answer["status"] != "planProposed" || answer["planId"] == "" || len(plans.plans) != 1 {
		t.Fatalf("the answer = %v, the stored plans %d", answer, len(plans.plans))
	}
	stored := plans.plans[0]
	if stored.ConversationID != "conv-1" || stored.SiteID != "site-1" || len(stored.Steps) != 2 || stored.Steps[1].DependsOn[0] != 0 ||
		stored.Steps[0].Summary == "" || stored.Status != agent.PlanProposed {
		t.Fatalf("the stored plan = %+v", stored)
	}
	if len(bus.types) != 1 || bus.types[0] != events.AgentPlanProposed {
		t.Errorf("the events = %v, want the plan proposed", bus.types)
	}

	got, err := planCall(t, registry, b, `{"op":"get","planId":"`+stored.ID+`"}`)
	if err != nil || got["plan"] == nil {
		t.Fatalf("get = %v, %v", got, err)
	}

	cases := map[string]string{
		"a step that only reads":           `{"op":"propose","title":"t","steps":[{"tool":"sites_get","args":"{}","reason":"r"}]}`,
		"a step its tool would refuse":     `{"op":"propose","title":"t","steps":[{"tool":"graph_add_edge","args":"{\"color\":\"red\"}","reason":"r"}]}`,
		"a tool nobody registered":         `{"op":"propose","title":"t","steps":[{"tool":"graph_paint","args":"{}","reason":"r"}]}`,
		"arguments that are no JSON":       `{"op":"propose","title":"t","steps":[{"tool":"graph_add_edge","args":"not json","reason":"r"}]}`,
		"a plan inside a plan":             `{"op":"propose","title":"t","steps":[{"tool":"plans","args":"{}","reason":"r"}]}`,
		"a dependency on a step not there": `{"op":"propose","title":"t","steps":[{"tool":"graph_move_entity","args":"{\"entityId\":\"a\",\"newParentId\":\"b\"}","reason":"r","dependsOn":[2]}]}`,
	}
	for name, args := range cases {
		if _, err := planCall(t, registry, b, args); err == nil {
			t.Errorf("%s was proposed", name)
		} else if code := errors.CodeOf(err); code != errors.Invalid && code != errors.NotFound {
			t.Errorf("%s answered %v", name, err)
		}
	}
	if len(plans.plans) != 1 {
		t.Errorf("a refused proposal stored a plan: %d", len(plans.plans))
	}
	if _, err := planCall(t, registry, tools.Binding{SiteID: "site-1"}, `{"op":"list"}`); err == nil || !strings.Contains(err.Error(), "conversation") {
		t.Errorf("a plan outside a conversation = %v", err)
	}
}
