package agent_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

var stamp = time.Date(2026, time.September, 18, 9, 0, 0, 0, time.UTC)

func TestNewConversationDefaultsToConfirmMode(t *testing.T) {
	t.Parallel()

	created, err := agent.NewConversation(agent.Conversation{ID: "c1", Title: "  plan   the   hub  ", CreatedAt: stamp})
	if err != nil {
		t.Fatalf("NewConversation: %v", err)
	}
	if created.Mode != agent.ModeConfirm {
		t.Fatalf("mode = %q, want confirm", created.Mode)
	}
	if created.Title != "plan the hub" {
		t.Fatalf("title = %q", created.Title)
	}
}

func TestNewConversationRefusesWhatItCannotStore(t *testing.T) {
	t.Parallel()

	empty := ""
	cases := []struct {
		name         string
		conversation agent.Conversation
	}{
		{name: "no id", conversation: agent.Conversation{CreatedAt: stamp}},
		{name: "unknown mode", conversation: agent.Conversation{ID: "c1", Mode: "maybe", CreatedAt: stamp}},
		{name: "empty site", conversation: agent.Conversation{ID: "c1", SiteID: &empty, CreatedAt: stamp}},
		{name: "no timestamp", conversation: agent.Conversation{ID: "c1"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := agent.NewConversation(tc.conversation); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewConversation = %v, want invalid", err)
			}
		})
	}
}

func TestTitleIsBoundedAndFolded(t *testing.T) {
	t.Parallel()

	long := strings.Repeat("a", agent.MaxTitle+40)
	if got := agent.Title(long); len([]rune(got)) != agent.MaxTitle {
		t.Fatalf("Title kept %d runes, want %d", len([]rune(got)), agent.MaxTitle)
	}
	if got := agent.Title("  one\n\ttwo  "); got != "one two" {
		t.Fatalf("Title = %q", got)
	}
}

func TestNewMessage(t *testing.T) {
	t.Parallel()

	message, err := agent.NewMessage(agent.Message{
		ID: "m1", ConversationID: "c1", Seq: 1, Role: agent.RoleUser, Text: "hello", CreatedAt: stamp,
	})
	if err != nil {
		t.Fatalf("NewMessage: %v", err)
	}
	if string(message.Payload) != "{}" {
		t.Fatalf("payload = %s, want an empty object", message.Payload)
	}

	cases := []struct {
		name    string
		message agent.Message
	}{
		{name: "no id", message: agent.Message{ConversationID: "c1", Seq: 1, Role: agent.RoleUser, CreatedAt: stamp}},
		{name: "no conversation", message: agent.Message{ID: "m1", Seq: 1, Role: agent.RoleUser, CreatedAt: stamp}},
		{name: "no sequence", message: agent.Message{ID: "m1", ConversationID: "c1", Role: agent.RoleUser, CreatedAt: stamp}},
		{name: "unknown role", message: agent.Message{ID: "m1", ConversationID: "c1", Seq: 1, Role: "oracle", CreatedAt: stamp}},
		{
			name:    "a tool message names no tool",
			message: agent.Message{ID: "m1", ConversationID: "c1", Seq: 1, Role: agent.RoleTool, CreatedAt: stamp},
		},
		{name: "no timestamp", message: agent.Message{ID: "m1", ConversationID: "c1", Seq: 1, Role: agent.RoleUser}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := agent.NewMessage(tc.message); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewMessage = %v, want invalid", err)
			}
		})
	}
}

func TestNewPendingAction(t *testing.T) {
	t.Parallel()

	action, err := agent.NewPendingAction(agent.PendingAction{
		ID: "a1", ConversationID: "c1", Tool: "runs.start", Summary: "start a run", CreatedAt: stamp,
	})
	if err != nil {
		t.Fatalf("NewPendingAction: %v", err)
	}
	if action.Status != agent.ActionPending || string(action.Args) != "{}" {
		t.Fatalf("action = %+v", action)
	}

	cases := []struct {
		name   string
		action agent.PendingAction
	}{
		{name: "no id", action: agent.PendingAction{ConversationID: "c1", Tool: "t", Summary: "s", CreatedAt: stamp}},
		{name: "no conversation", action: agent.PendingAction{ID: "a1", Tool: "t", Summary: "s", CreatedAt: stamp}},
		{name: "no tool", action: agent.PendingAction{ID: "a1", ConversationID: "c1", Summary: "s", CreatedAt: stamp}},
		{name: "no summary", action: agent.PendingAction{ID: "a1", ConversationID: "c1", Tool: "t", CreatedAt: stamp}},
		{
			name:   "unknown status",
			action: agent.PendingAction{ID: "a1", ConversationID: "c1", Tool: "t", Summary: "s", Status: "later", CreatedAt: stamp},
		},
		{name: "no timestamp", action: agent.PendingAction{ID: "a1", ConversationID: "c1", Tool: "t", Summary: "s"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := agent.NewPendingAction(tc.action); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewPendingAction = %v, want invalid", err)
			}
		})
	}
}

func TestActionStatusSettles(t *testing.T) {
	t.Parallel()

	settled := map[agent.ActionStatus]bool{
		agent.ActionPending:  false,
		agent.ActionApproved: false,
		agent.ActionRejected: true,
		agent.ActionExecuted: true,
		agent.ActionFailed:   true,
	}
	for status, want := range settled {
		if !status.Valid() {
			t.Fatalf("%q is not a known action status", status)
		}
		if got := status.Settled(); got != want {
			t.Errorf("%q settled = %t, want %t", status, got, want)
		}
	}
	if agent.ActionStatus("later").Valid() {
		t.Error("an unknown action status must not validate")
	}
}

func TestNewToolCall(t *testing.T) {
	t.Parallel()

	call, err := agent.NewToolCall(agent.ToolCall{
		ID: "t1", ConversationID: "c1", Tool: "sites.list", Status: agent.CallOK,
		Args: json.RawMessage(`{"limit":5}`), CreatedAt: stamp,
	})
	if err != nil {
		t.Fatalf("NewToolCall: %v", err)
	}
	if string(call.Args) != `{"limit":5}` {
		t.Fatalf("args = %s", call.Args)
	}

	cases := []struct {
		name string
		call agent.ToolCall
	}{
		{name: "no id", call: agent.ToolCall{ConversationID: "c1", Tool: "t", Status: agent.CallOK, CreatedAt: stamp}},
		{name: "no conversation", call: agent.ToolCall{ID: "t1", Tool: "t", Status: agent.CallOK, CreatedAt: stamp}},
		{name: "no tool", call: agent.ToolCall{ID: "t1", ConversationID: "c1", Status: agent.CallOK, CreatedAt: stamp}},
		{name: "unknown status", call: agent.ToolCall{ID: "t1", ConversationID: "c1", Tool: "t", Status: "maybe", CreatedAt: stamp}},
		{
			name: "negative duration",
			call: agent.ToolCall{ID: "t1", ConversationID: "c1", Tool: "t", Status: agent.CallOK, DurationMS: -1, CreatedAt: stamp},
		},
		{name: "no timestamp", call: agent.ToolCall{ID: "t1", ConversationID: "c1", Tool: "t", Status: agent.CallOK}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := agent.NewToolCall(tc.call); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("NewToolCall = %v, want invalid", err)
			}
		})
	}
}

func TestModeAndRoleValidate(t *testing.T) {
	t.Parallel()

	for _, mode := range []agent.Mode{agent.ModeConfirm, agent.ModeAutonomous} {
		if !mode.Valid() {
			t.Errorf("%q is a known mode", mode)
		}
	}
	if agent.Mode("silent").Valid() {
		t.Error("an unknown mode must not validate")
	}

	for _, role := range []agent.Role{agent.RoleUser, agent.RoleAssistant, agent.RoleTool} {
		if !role.Valid() {
			t.Errorf("%q is a known role", role)
		}
	}
	if agent.Role("oracle").Valid() {
		t.Error("an unknown role must not validate")
	}
	if !agent.CallDenied.Valid() || agent.CallStatus("maybe").Valid() {
		t.Error("the call statuses are ok, denied and error")
	}
}

func TestSuggestedTitle(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		want string
		ok   bool
	}{
		{name: "a plain answer is kept", raw: "Plan the hub pages", want: "Plan the hub pages", ok: true},
		{name: "surrounding straight quotes go", raw: "\"Plan the hub pages\"", want: "Plan the hub pages", ok: true},
		{name: "surrounding curly quotes go", raw: "“Plan the hub pages”", want: "Plan the hub pages", ok: true},
		{name: "a trailing period goes", raw: "Plan the hub pages.", want: "Plan the hub pages", ok: true},
		{name: "a label prefix goes", raw: "Title: Plan the hub pages", want: "Plan the hub pages", ok: true},
		{name: "a lowercase label prefix goes", raw: "title: Plan the hub", want: "Plan the hub", ok: true},
		{name: "whitespace collapses", raw: "  Plan   the\n hub  ", want: "Plan the hub", ok: true},
		{name: "quotes and a period together", raw: "\"Plan the hub.\"", want: "Plan the hub", ok: true},
		{name: "empty is refused", raw: "", ok: false},
		{name: "blank is refused", raw: "   \n ", ok: false},
		{name: "punctuation alone is refused", raw: "\".\"", ok: false},
		{name: "a rambling answer is refused", raw: strings.Repeat("word ", 20), ok: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, ok := agent.SuggestedTitle(tc.raw)
			if ok != tc.ok {
				t.Fatalf("SuggestedTitle(%q) usable = %t, want %t", tc.raw, ok, tc.ok)
			}
			if ok && got != tc.want {
				t.Errorf("SuggestedTitle(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}
}
