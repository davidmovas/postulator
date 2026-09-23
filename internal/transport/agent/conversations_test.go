package agent_test

import (
	stderrors "errors"
	"testing"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/events"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func TestRenameConversationKeepsTheTitleTidy(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeConfirm)

	renamed, err := h.service.RenameConversation(t.Context(), agentapp.RenameConversationRequest{
		ConversationID: conversation, Title: "  Build   the guide ",
	})
	if err != nil || renamed.Conversation.Title != "Build the guide" {
		t.Fatalf("RenameConversation = %+v, %v", renamed, err)
	}

	listed, err := h.service.ListConversations(t.Context(), agentapp.ListConversationsRequest{SiteID: h.siteID})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].Title != "Build the guide" {
		t.Fatalf("ListConversations after the rename = %+v, %v", listed, err)
	}

	cases := []struct {
		name    string
		request agentapp.RenameConversationRequest
		code    errors.Code
		field   string
	}{
		{
			name:    "a blank title is refused",
			request: agentapp.RenameConversationRequest{ConversationID: conversation, Title: "   "},
			code:    errors.Invalid,
			field:   "title",
		},
		{
			name:    "a conversation is needed",
			request: agentapp.RenameConversationRequest{Title: "anything"},
			code:    errors.Invalid,
			field:   "conversationId",
		},
		{
			name:    "an unknown conversation is missing",
			request: agentapp.RenameConversationRequest{ConversationID: id.New(), Title: "anything"},
			code:    errors.NotFound,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, renameErr := h.service.RenameConversation(t.Context(), tc.request)
			if !errors.IsCode(renameErr, tc.code) {
				t.Fatalf("RenameConversation = %v, want %s", renameErr, tc.code)
			}
			if tc.field != "" && fieldOf(renameErr) != tc.field {
				t.Fatalf("RenameConversation names the field %q, want %q", fieldOf(renameErr), tc.field)
			}
		})
	}
}

func fieldOf(err error) string {
	var kernel *errors.Error
	if !stderrors.As(err, &kernel) {
		return ""
	}
	field, ok := kernel.Details["field"].(string)
	if !ok {
		return ""
	}
	return field
}

func TestTheFirstMessageTitlesAnUntitledConversationAtOnce(t *testing.T) {
	t.Parallel()

	service, conversation, _ := blocked(t)

	if _, err := service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "  Which   pages carry\nno entity? ",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	listed, err := service.ListConversations(t.Context(), agentapp.ListConversationsRequest{})
	if err != nil || len(listed.Items) != 1 {
		t.Fatalf("ListConversations = %+v, %v", listed, err)
	}
	if listed.Items[0].Title != "Which pages carry no entity?" {
		t.Fatalf("the title while the turn runs is %q", listed.Items[0].Title)
	}
}

func TestTheFinishedTurnNamesTheConversation(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.titler.script("Pages without an entity", nil)

	created, err := h.service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{
		SiteID: h.siteID, Mode: string(domainagent.ModeAutonomous),
	})
	if err != nil || created.Conversation.Title != "" {
		t.Fatalf("CreateConversation = %+v, %v", created, err)
	}

	h.send(t, created.Conversation.ID, "Which pages carry no entity?")
	h.titledConversation(t)

	if got := h.titleOf(t, created.Conversation.ID); got != "Pages without an entity" {
		t.Fatalf("the conversation is named %q", got)
	}

	payload, ok := h.payload(events.AgentTitled).(events.AgentTitledPayload)
	if !ok || payload.ConversationID != created.Conversation.ID || payload.Title != "Pages without an entity" {
		t.Fatalf("agent.titled carried %+v", h.payload(events.AgentTitled))
	}
}

func TestAConversationIsNamedOnceAndNeverAgain(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.titler.script("", errors.New(errors.External, "the titler is down"))

	created, err := h.service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{
		SiteID: h.siteID, Mode: string(domainagent.ModeAutonomous),
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	h.send(t, created.Conversation.ID, "Which pages carry no entity?")
	h.titledConversation(t)

	if got := h.titleOf(t, created.Conversation.ID); got != "Which pages carry no entity?" {
		t.Fatalf("a refused title left %q", got)
	}

	h.titler.script("A second chance the turn must not take", nil)
	h.send(t, created.Conversation.ID, "And which of them are drafts?")

	if got := h.titleOf(t, created.Conversation.ID); got != "Which pages carry no entity?" {
		t.Fatalf("the second turn renamed the conversation to %q", got)
	}
	if attempts := h.titler.attempts(); attempts != 1 {
		t.Fatalf("the titler was asked %d times, want 1", attempts)
	}
}

func TestANamedConversationIsNeverRenamedByTheTurn(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.titler.script("A name the client did not choose", nil)

	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "Which pages carry no entity?")

	if got := h.titleOf(t, conversation); got != "plan the hub" {
		t.Fatalf("the named conversation became %q", got)
	}
	if attempts := h.titler.attempts(); attempts != 0 {
		t.Fatalf("the titler was asked %d times for a named conversation", attempts)
	}
}

func TestDeleteConversationStopsTheTurnAndTakesTheRows(t *testing.T) {
	t.Parallel()

	service, conversation, runner := blocked(t)

	if _, err := service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "the first question",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	<-runner.entered

	if _, err := service.DeleteConversation(t.Context(), agentapp.DeleteConversationRequest{
		ConversationID: conversation,
	}); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}

	if _, err := service.SetMode(t.Context(), agentapp.SetModeRequest{
		ConversationID: conversation, Mode: string(domainagent.ModeConfirm),
	}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("SetMode after the deletion = %v", err)
	}
	listed, err := service.ListMessages(t.Context(), agentapp.ListMessagesRequest{ConversationID: conversation})
	if err != nil || len(listed.Items) != 0 {
		t.Fatalf("the transcript survived the deletion: %+v, %v", listed.Items, err)
	}
	if _, err = service.DeleteConversation(t.Context(), agentapp.DeleteConversationRequest{
		ConversationID: conversation,
	}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("a second DeleteConversation = %v", err)
	}
	if _, err = service.DeleteConversation(t.Context(), agentapp.DeleteConversationRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("DeleteConversation without a conversation = %v", err)
	}
}
