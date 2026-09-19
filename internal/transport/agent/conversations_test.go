package agent_test

import (
	stderrors "errors"
	"testing"

	agentapp "github.com/davidmovas/postulator/internal/application/agent"
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

func TestTheFirstMessageTitlesAnUntitledConversation(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{
		SiteID: h.siteID, Mode: string(domainagent.ModeAutonomous),
	})
	if err != nil || created.Conversation.Title != "" {
		t.Fatalf("CreateConversation = %+v, %v", created, err)
	}
	titled := h.conversation(t, domainagent.ModeAutonomous)

	h.send(t, created.Conversation.ID, "  Which   pages carry\nno entity? ")
	h.send(t, titled, "  Which   pages carry no entity? ")

	listed, err := h.service.ListConversations(t.Context(), agentapp.ListConversationsRequest{SiteID: h.siteID})
	if err != nil || len(listed.Items) != 2 {
		t.Fatalf("ListConversations = %+v, %v", listed, err)
	}
	titles := map[string]string{}
	for _, item := range listed.Items {
		titles[item.ID] = item.Title
	}
	if titles[created.Conversation.ID] != "Which pages carry no entity?" {
		t.Fatalf("the untitled conversation is now %q", titles[created.Conversation.ID])
	}
	if titles[titled] != "plan the hub" {
		t.Fatalf("the titled conversation became %q", titles[titled])
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
