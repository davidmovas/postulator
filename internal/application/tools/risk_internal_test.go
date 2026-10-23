package tools

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/clock"
)

type heldActions struct {
	actions []agent.PendingAction
}

func (h *heldActions) Insert(_ context.Context, action agent.PendingAction) error {
	h.actions = append(h.actions, action)
	return nil
}

type heldEvents struct {
	payloads []events.AgentConfirmRequestedPayload
}

func (h *heldEvents) Publish(_ events.Type, payload any) error {
	if asked, ok := payload.(events.AgentConfirmRequestedPayload); ok {
		h.payloads = append(h.payloads, asked)
	}
	return nil
}

type opArgs struct {
	Op    string `json:"op" enum:"list,set"`
	Value string `json:"value,omitempty"`
}

type foldedTool struct {
	ran      int
	refuse   bool
	summary  string
	registry *Registry
	actions  *heldActions
	events   *heldEvents
}

func newFoldedTool(t *testing.T) *foldedTool {
	t.Helper()

	folded := &foldedTool{summary: "Set the value on the site", actions: &heldActions{}, events: &heldEvents{}}
	build := func(Deps) Tool {
		tool := confirming(NewTool(Def{Name: "things_fold", Description: "Read or set a thing.", Risk: RiskWrite},
			func(context.Context, Binding, opArgs) (string, error) {
				folded.ran++
				return "done", nil
			}), func(_ context.Context, _ Binding, in opArgs) (string, error) {
			if folded.refuse {
				return "", stderrors.New("the page cannot be read")
			}
			if in.Op == "set" {
				return folded.summary, nil
			}
			return "", nil
		})
		return readsOn(tool, "list")
	}
	deps := Deps{Actions: folded.actions, Publisher: folded.events, Clock: clock.NewFake(time.Date(2026, 10, 23, 9, 0, 0, 0, time.UTC))}
	folded.registry = &Registry{deps: deps, byName: map[string]factory{"things_fold": build}, order: []string{"things_fold"}}
	return folded
}

func TestACallsRiskComesFromItsArguments(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		mode     agent.Mode
		approved bool
		op       string
		summary  string
		ran      int
		proposed int
	}{
		{name: "a read op in confirm mode runs", mode: agent.ModeConfirm, op: "list", ran: 1},
		{name: "a read op in an autonomous conversation runs", mode: agent.ModeAutonomous, op: "list", ran: 1},
		{name: "a write op in confirm mode waits", mode: agent.ModeConfirm, op: "set", summary: "Set the value on the site", proposed: 1},
		{name: "a write op with a summary waits in an autonomous conversation too", mode: agent.ModeAutonomous, op: "set", summary: "Set the value on the site", proposed: 1},
		{name: "an approved write op runs", mode: agent.ModeConfirm, approved: true, op: "set", ran: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, path := range []string{"build", "call"} {
				folded := newFoldedTool(t)
				binding := Binding{SiteID: "site", ConversationID: "conversation", Mode: tc.mode, Approved: tc.approved}
				args := json.RawMessage(`{"op":"` + tc.op + `"}`)
				var err error
				if path == "build" {
					_, err = folded.registry.Build(binding)[0].Run(t.Context(), binding, args)
				} else {
					_, err = folded.registry.Call(t.Context(), binding, "things_fold", args)
				}
				if err != nil {
					t.Fatalf("%s: %v", path, err)
				}
				if folded.ran != tc.ran || len(folded.actions.actions) != tc.proposed {
					t.Fatalf("%s: the tool ran %d times and proposed %d, want %d and %d",
						path, folded.ran, len(folded.actions.actions), tc.ran, tc.proposed)
				}
				if tc.proposed == 0 {
					continue
				}
				if got := folded.actions.actions[0].Summary; got != tc.summary {
					t.Errorf("%s: the proposal says %q, want %q", path, got, tc.summary)
				}
				if len(folded.events.payloads) != 1 || folded.events.payloads[0].Risk != string(RiskWrite) {
					t.Errorf("%s: the card carries %+v, want the write risk of this call", path, folded.events.payloads)
				}
			}
		})
	}
}

func TestAWriteWhoseSummaryCannotBeBuiltIsRefused(t *testing.T) {
	t.Parallel()

	for _, mode := range []agent.Mode{agent.ModeConfirm, agent.ModeAutonomous} {
		folded := newFoldedTool(t)
		folded.refuse = true
		binding := Binding{SiteID: "site", ConversationID: "conversation", Mode: mode}
		if _, err := folded.registry.Call(t.Context(), binding, "things_fold", json.RawMessage(`{"op":"set"}`)); err == nil {
			t.Errorf("%s: a write whose summary failed was not refused", mode)
		}
		if folded.ran != 0 || len(folded.actions.actions) != 0 {
			t.Errorf("%s: the tool ran %d times and proposed %d, want neither", mode, folded.ran, len(folded.actions.actions))
		}
	}
}

func TestAWriteWithNoSummaryOfItsOwnFollowsTheMode(t *testing.T) {
	t.Parallel()

	folded := newFoldedTool(t)
	folded.summary = ""
	binding := Binding{SiteID: "site", ConversationID: "conversation", Mode: agent.ModeAutonomous}
	if _, err := folded.registry.Call(t.Context(), binding, "things_fold", json.RawMessage(`{"op":"set"}`)); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if folded.ran != 1 || len(folded.actions.actions) != 0 {
		t.Fatalf("an autonomous write with no summary ran %d times and proposed %d, want it run as any autonomous write",
			folded.ran, len(folded.actions.actions))
	}
	binding.Mode = agent.ModeConfirm
	if _, err := folded.registry.Call(t.Context(), binding, "things_fold", json.RawMessage(`{"op":"set"}`)); err != nil {
		t.Fatalf("Call: %v", err)
	}
	if len(folded.actions.actions) != 1 || folded.actions.actions[0].Summary == "" {
		t.Fatalf("a confirm-mode write with no summary proposed %+v, want the generic proposal", folded.actions.actions)
	}
}
