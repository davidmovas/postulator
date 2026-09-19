package sqlite_test

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type agentFixture struct {
	store        *sqlite.Store
	conversation *sqlite.ConversationRepo
	messages     *sqlite.MessageRepo
	actions      *sqlite.PendingActionRepo
	calls        *sqlite.ToolCallRepo
	histories    *sqlite.ConversationHistoryRepo
	siteID       string
}

func newAgentFixture(t *testing.T) agentFixture {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	return agentFixture{
		store:        store,
		conversation: sqlite.NewConversationRepo(store),
		messages:     sqlite.NewMessageRepo(store),
		actions:      sqlite.NewPendingActionRepo(store),
		calls:        sqlite.NewToolCallRepo(store),
		histories:    sqlite.NewConversationHistoryRepo(store),
		siteID:       owner.ID,
	}
}

func (f agentFixture) conversationOf(t *testing.T, mode agent.Mode) agent.Conversation {
	t.Helper()

	record, err := agent.NewConversation(agent.Conversation{
		ID: id.New(), SiteID: &f.siteID, Title: "plan the hub", Mode: mode,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewConversation: %v", err)
	}
	if insertErr := f.conversation.Insert(t.Context(), record); insertErr != nil {
		t.Fatalf("insert the conversation: %v", insertErr)
	}
	return record
}

func TestConversationRoundTrip(t *testing.T) {
	t.Parallel()

	f := newAgentFixture(t)
	record := f.conversationOf(t, agent.ModeConfirm)

	read, err := f.conversation.Get(t.Context(), record.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if read.SiteID == nil || *read.SiteID != f.siteID || read.Mode != agent.ModeConfirm {
		t.Fatalf("conversation = %+v", read)
	}

	read.Mode = agent.ModeAutonomous
	read.Title = "build the guide"
	read.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = f.conversation.Update(t.Context(), read); err != nil {
		t.Fatalf("Update: %v", err)
	}

	again, err := f.conversation.Get(t.Context(), record.ID)
	if err != nil || again.Mode != agent.ModeAutonomous || again.Title != "build the guide" {
		t.Fatalf("conversation after update = %+v, %v", again, err)
	}

	listed, err := f.conversation.List(t.Context(), agent.ConversationQuery{SiteID: f.siteID}, paging.Request{Limit: 10})
	if err != nil || len(listed.Items) != 1 {
		t.Fatalf("List = %+v, %v", listed, err)
	}

	if _, err = f.conversation.Get(t.Context(), id.New()); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get of an unknown conversation = %v", err)
	}
	missing := again
	missing.ID = id.New()
	if err = f.conversation.Update(t.Context(), missing); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Update of an unknown conversation = %v", err)
	}
	if err = f.conversation.Insert(t.Context(), record); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Insert of a duplicate = %v", err)
	}
}

func TestMessagesAreNumberedPerConversation(t *testing.T) {
	t.Parallel()

	f := newAgentFixture(t)
	first := f.conversationOf(t, agent.ModeConfirm)
	second := f.conversationOf(t, agent.ModeConfirm)

	for _, conversationID := range []string{first.ID, first.ID, second.ID} {
		message, err := agent.NewMessage(agent.Message{
			ID: id.New(), ConversationID: conversationID, Seq: 1, Role: agent.RoleUser, Text: "hello",
			CreatedAt: sqlitetest.Stamp,
		})
		if err != nil {
			t.Fatalf("NewMessage: %v", err)
		}
		stored, appendErr := f.messages.Append(t.Context(), message)
		if appendErr != nil {
			t.Fatalf("Append: %v", appendErr)
		}
		if stored.Seq == 0 {
			t.Fatalf("stored message = %+v", stored)
		}
	}

	rows, err := f.messages.ByConversation(t.Context(), first.ID)
	if err != nil || len(rows) != 2 || rows[0].Seq != 1 || rows[1].Seq != 2 {
		t.Fatalf("ByConversation = %+v, %v", rows, err)
	}

	latest, err := f.messages.LatestSeq(t.Context(), first.ID)
	if err != nil || latest != 2 {
		t.Fatalf("LatestSeq = %d, %v", latest, err)
	}
	empty, err := f.messages.LatestSeq(t.Context(), id.New())
	if err != nil || empty != 0 {
		t.Fatalf("LatestSeq of an empty conversation = %d, %v", empty, err)
	}

	listed, err := f.messages.List(t.Context(), agent.MessageQuery{ConversationID: first.ID}, paging.Request{Limit: 1})
	if err != nil || len(listed.Items) != 1 || !listed.HasMore {
		t.Fatalf("List = %+v, %v", listed, err)
	}

	if _, err = f.messages.Get(t.Context(), id.New()); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get of an unknown message = %v", err)
	}
}

func TestPendingActionTransitionsOnce(t *testing.T) {
	t.Parallel()

	f := newAgentFixture(t)
	conversation := f.conversationOf(t, agent.ModeConfirm)

	action, err := agent.NewPendingAction(agent.PendingAction{
		ID: id.New(), ConversationID: conversation.ID, Tool: "runs.start", Summary: "start a run",
		Args: json.RawMessage(`{"siteId":"s1"}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewPendingAction: %v", err)
	}
	if err = f.actions.Insert(t.Context(), action); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	pending := agent.ActionPending
	listed, err := f.actions.List(t.Context(), agent.ActionQuery{ConversationID: conversation.ID, Status: &pending},
		paging.Request{Limit: 10})
	if err != nil || len(listed.Items) != 1 {
		t.Fatalf("List = %+v, %v", listed, err)
	}

	at := sqlitetest.Stamp.Add(time.Minute)
	taken, err := f.actions.Transition(t.Context(), action.ID, agent.ActionPending, agent.ActionExecuted,
		json.RawMessage(`{"runId":"r1"}`), "", at)
	if err != nil || !taken {
		t.Fatalf("Transition = %t, %v", taken, err)
	}

	again, err := f.actions.Transition(t.Context(), action.ID, agent.ActionPending, agent.ActionRejected, nil, "", at)
	if err != nil || again {
		t.Fatalf("a second transition from pending = %t, %v", again, err)
	}

	settled, err := f.actions.Get(t.Context(), action.ID)
	if err != nil || settled.Status != agent.ActionExecuted || string(settled.Result) != `{"runId":"r1"}` {
		t.Fatalf("action = %+v, %v", settled, err)
	}
	if _, err = f.actions.Get(t.Context(), id.New()); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get of an unknown action = %v", err)
	}
}

func TestToolCallsAndHistoryAreStoredPerConversation(t *testing.T) {
	t.Parallel()

	f := newAgentFixture(t)
	conversation := f.conversationOf(t, agent.ModeAutonomous)

	call, err := agent.NewToolCall(agent.ToolCall{
		ID: id.New(), ConversationID: conversation.ID, CallID: "call-1", Tool: "sites.list",
		Args: json.RawMessage(`{"limit":5}`), Status: agent.CallOK, DurationMS: 12, CreatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewToolCall: %v", err)
	}
	if err = f.calls.Insert(t.Context(), call); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	rows, err := f.calls.ByConversation(t.Context(), conversation.ID)
	if err != nil || len(rows) != 1 || rows[0].Tool != "sites.list" || rows[0].DurationMS != 12 {
		t.Fatalf("ByConversation = %+v, %v", rows, err)
	}

	if _, _, err = f.histories.Load(t.Context(), conversation.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Load of an empty history = %v", err)
	}

	if err = f.histories.Save(t.Context(), conversation.ID, []byte(`{"version":3}`), 3, sqlitetest.Stamp); err != nil {
		t.Fatalf("Save: %v", err)
	}
	next := []byte(`{"version":3,"messages":[]}`)
	if err = f.histories.Save(t.Context(), conversation.ID, next, 3, sqlitetest.Stamp.Add(time.Minute)); err != nil {
		t.Fatalf("Save again: %v", err)
	}

	read, version, err := f.histories.Load(t.Context(), conversation.ID)
	if err != nil || !bytes.Equal(read, next) || version != 3 {
		t.Fatalf("Load = %s, %d, %v", read, version, err)
	}
}

func TestDeletingAConversationTakesItsRows(t *testing.T) {
	t.Parallel()

	f := newAgentFixture(t)
	conversation := f.conversationOf(t, agent.ModeConfirm)
	kept := f.conversationOf(t, agent.ModeConfirm)

	for _, conversationID := range []string{conversation.ID, kept.ID} {
		message, err := agent.NewMessage(agent.Message{
			ID: id.New(), ConversationID: conversationID, Seq: 1, Role: agent.RoleUser, Text: "hello",
			CreatedAt: sqlitetest.Stamp,
		})
		if err != nil {
			t.Fatalf("NewMessage: %v", err)
		}
		if _, err = f.messages.Append(t.Context(), message); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}
	action, err := agent.NewPendingAction(agent.PendingAction{
		ID: id.New(), ConversationID: conversation.ID, Tool: "runs.start", Summary: "start a run",
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewPendingAction: %v", err)
	}
	if err = f.actions.Insert(t.Context(), action); err != nil {
		t.Fatalf("Insert the action: %v", err)
	}
	if err = f.histories.Save(t.Context(), conversation.ID, []byte(`{"version":3}`), 3, sqlitetest.Stamp); err != nil {
		t.Fatalf("Save the history: %v", err)
	}

	if err = f.conversation.Delete(t.Context(), conversation.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	if _, err = f.conversation.Get(t.Context(), conversation.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("the conversation survived its deletion: %v", err)
	}
	rows, err := f.messages.ByConversation(t.Context(), conversation.ID)
	if err != nil || len(rows) != 0 {
		t.Fatalf("the messages survived the conversation: %+v, %v", rows, err)
	}
	if _, err = f.actions.Get(t.Context(), action.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("the action survived the conversation: %v", err)
	}
	if _, _, err = f.histories.Load(t.Context(), conversation.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("the history survived the conversation: %v", err)
	}
	others, err := f.messages.ByConversation(t.Context(), kept.ID)
	if err != nil || len(others) != 1 {
		t.Fatalf("the other conversation lost its messages: %+v, %v", others, err)
	}
	if err = f.conversation.Delete(t.Context(), conversation.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Delete of a deleted conversation = %v", err)
	}
}

func TestDeletingASiteTakesItsConversations(t *testing.T) {
	t.Parallel()

	f := newAgentFixture(t)
	conversation := f.conversationOf(t, agent.ModeConfirm)

	if err := sqlite.NewSiteRepo(f.store).Delete(t.Context(), f.siteID); err != nil {
		t.Fatalf("delete the site: %v", err)
	}
	if _, err := f.conversation.Get(t.Context(), conversation.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("the conversation survived the site: %v", err)
	}
}
