package agent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/gollem-dev/gollem"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/paging"
	agentrunner "github.com/davidmovas/postulator/internal/transport/agent"
)

const historyFenceSlack = 64

func TestATurnStreamsItsAnswerAndRecordsTheSpend(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "FAKE: the hub is planned")

	seen := h.seen()
	delta := slices.Index(seen, events.AgentDelta)
	done := slices.Index(seen, events.AgentDone)
	if delta < 0 || done < 0 || delta > done {
		t.Fatalf("the bus saw %v, want a delta before the done", seen)
	}

	finished, ok := h.payload(events.AgentDone).(events.AgentDonePayload)
	if !ok || finished.Text != "the hub is planned" || finished.Error != "" {
		t.Fatalf("the done payload is %+v", h.payload(events.AgentDone))
	}
	if finished.InputTokens == 0 || finished.OutputTokens == 0 {
		t.Fatalf("the turn reported no usage: %+v", finished)
	}

	stored := h.messages(t, conversation)
	if len(stored) != 2 || stored[0].Role != "user" || stored[1].Role != "assistant" {
		t.Fatalf("the transcript is %+v", stored)
	}
	if stored[1].ID != finished.MessageID || stored[1].Text != "the hub is planned" {
		t.Fatalf("the answer is %+v", stored[1])
	}

	spend, err := h.calls.SumByConversation(t.Context(), conversation)
	if err != nil || spend.Calls != 1 || spend.Usage.Total == 0 {
		t.Fatalf("the ledger holds %+v, %v", spend, err)
	}
}

func TestAToolCallIsFencedAuditedAndKept(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "TOOL:pages_tree{}\nFAKE: the tree is empty")

	seen := h.seen()
	started := slices.Index(seen, events.AgentToolStarted)
	finished := slices.Index(seen, events.AgentToolFinished)
	done := slices.Index(seen, events.AgentDone)
	if started < 0 || finished < started || done < finished {
		t.Fatalf("the bus saw %v, want started, finished and then done", seen)
	}

	outcome, ok := h.payload(events.AgentToolFinished).(events.AgentToolFinishedPayload)
	if !ok || outcome.Tool != "pages_tree" || outcome.Status != string(domainagent.CallOK) {
		t.Fatalf("the finished payload is %+v", h.payload(events.AgentToolFinished))
	}

	recorded, err := sqlite.NewToolCallRepo(h.store).ByConversation(t.Context(), conversation)
	if err != nil || len(recorded) != 1 || recorded[0].Tool != "pages_tree" {
		t.Fatalf("the tool call ledger holds %+v, %v", recorded, err)
	}

	stored := h.messages(t, conversation)
	if len(stored) != 3 || stored[1].Role != "tool" || stored[1].Tool != "pages_tree" {
		t.Fatalf("the transcript is %+v", stored)
	}

	sessions := h.model.Sessions()
	if len(sessions) != 1 {
		t.Fatalf("the model opened %d sessions", len(sessions))
	}
	history, err := sessions[0].History()
	if err != nil {
		t.Fatalf("read the model history: %v", err)
	}

	fenced := false
	for _, message := range history.Messages {
		for _, content := range message.Contents {
			if strings.Contains(string(content.Data), "untrustedContent") {
				fenced = true
			}
		}
	}
	if !fenced {
		t.Fatal("the tool result reached the model without being fenced as untrusted data")
	}
}

func TestConfirmModeProposesAndTheApprovalSurvivesARestart(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	doomed := sqlitetest.Page(t, h.store, h.siteID, "/coffee/doomed/")
	conversation := h.conversation(t, domainagent.ModeConfirm)

	h.send(t, conversation, `TOOL:pages_delete{"id":"`+doomed.ID+`"}`+"\nFAKE: waiting for you")

	requested, ok := h.payload(events.AgentConfirmRequested).(events.AgentConfirmRequestedPayload)
	if !ok || requested.Tool != "pages_delete" || requested.ConversationID != conversation {
		t.Fatalf("the confirmation payload is %+v", h.payload(events.AgentConfirmRequested))
	}
	if requested.Summary == "" || requested.Risk != "dangerous" {
		t.Fatalf("the confirmation is %+v", requested)
	}

	if _, err := h.pages.Get(t.Context(), doomed.ID); err != nil {
		t.Fatalf("a proposed delete must not have run: %v", err)
	}

	pending, err := h.service.ListPendingActions(t.Context(), agentapp.ListPendingActionsRequest{
		ConversationID: conversation, Status: string(domainagent.ActionPending),
	})
	if err != nil || len(pending.Items) != 1 {
		t.Fatalf("the pending actions are %+v, %v", pending, err)
	}

	restarted := build(t, h.store, h.model, &applicationtest.Recorder{})
	restarted.siteID = h.siteID

	confirmed, err := restarted.service.Confirm(t.Context(), agentapp.ConfirmRequest{
		ActionID: pending.Items[0].ID, Approve: true,
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if confirmed.Action.Status != string(domainagent.ActionExecuted) {
		t.Fatalf("the action settled as %+v", confirmed.Action)
	}

	if _, err = h.pages.Get(t.Context(), doomed.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("the approved delete did not run: %v", err)
	}

	resolved, ok := restarted.payload(events.AgentConfirmResolved).(events.AgentConfirmResolvedPayload)
	if !ok || resolved.Status != string(domainagent.ActionExecuted) || resolved.Tool != "pages_delete" {
		t.Fatalf("the resolved payload is %+v", restarted.payload(events.AgentConfirmResolved))
	}

	again, err := restarted.service.Confirm(t.Context(), agentapp.ConfirmRequest{
		ActionID: pending.Items[0].ID, Approve: true,
	})
	if !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("a second confirmation = %+v, %v; want a conflict", again, err)
	}
}

func TestARejectedConfirmationNeverRuns(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	doomed := sqlitetest.Page(t, h.store, h.siteID, "/coffee/spared/")
	conversation := h.conversation(t, domainagent.ModeConfirm)
	h.send(t, conversation, `TOOL:pages_delete{"id":"`+doomed.ID+`"}`+"\nFAKE: waiting for you")

	pending, err := h.service.ListPendingActions(t.Context(), agentapp.ListPendingActionsRequest{
		ConversationID: conversation,
	})
	if err != nil || len(pending.Items) != 1 {
		t.Fatalf("the pending actions are %+v, %v", pending, err)
	}

	rejected, err := h.service.Confirm(t.Context(), agentapp.ConfirmRequest{ActionID: pending.Items[0].ID})
	if err != nil || rejected.Action.Status != string(domainagent.ActionRejected) {
		t.Fatalf("Confirm = %+v, %v", rejected, err)
	}
	if _, err = h.pages.Get(t.Context(), doomed.ID); err != nil {
		t.Fatalf("a rejected delete must not have run: %v", err)
	}
}

func TestOneTurnAtATimePerConversation(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)

	turns := agentapp.NewTurns(nil)
	release := make(chan struct{})
	if err := turns.Start(t.Context(), conversation, "assistant-1", time.Now(),
		func(context.Context) { <-release }); err != nil {
		t.Fatalf("Start: %v", err)
	}
	if err := turns.Start(t.Context(), conversation, "assistant-2", time.Now(),
		func(context.Context) {}); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("a second turn = %v, want a conflict", err)
	}
	if !turns.Cancel(conversation) {
		t.Fatal("the running turn could not be cancelled")
	}

	close(release)
	turns.Close()
	if turns.Running(conversation) {
		t.Fatal("the turn is still running after Close")
	}
}

func TestTheModelFailureIsReportedAndTheLedgerRecordsIt(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "ERROR:EXTERNAL")

	finished, ok := h.payload(events.AgentDone).(events.AgentDonePayload)
	if !ok || finished.Error == "" {
		t.Fatalf("the done payload is %+v", h.payload(events.AgentDone))
	}

	listed, err := h.calls.List(t.Context(), domainllm.CallQuery{ConversationID: conversation}, paging.Request{Limit: 10})
	if err != nil || len(listed.Items) != 1 || listed.Items[0].Status != domainllm.CallError {
		t.Fatalf("the ledger holds %+v, %v", listed.Items, err)
	}
	if listed.Items[0].Step != agentrunner.ChatStep || listed.Items[0].ErrorCode == "" {
		t.Fatalf("the failed round reads %+v", listed.Items[0])
	}
}

func TestEveryModelCallOfATurnIsItsOwnLedgerRow(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "TOOL:pages_tree{}\nTOOL:pages_list{}\nFAKE: two reads and an answer")

	listed, err := h.calls.List(t.Context(), domainllm.CallQuery{ConversationID: conversation},
		paging.Request{Limit: 50})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(listed.Items) != 3 {
		t.Fatalf("the ledger holds %d rows for a turn of three model calls", len(listed.Items))
	}
	for _, call := range listed.Items {
		if call.Step != agentrunner.ChatStep || call.Status != domainllm.CallOK {
			t.Fatalf("a round reads %+v", call)
		}
		if call.ConversationID != conversation || call.Usage.Total == 0 {
			t.Fatalf("a round reads %+v", call)
		}
	}

	finished, ok := h.payload(events.AgentDone).(events.AgentDonePayload)
	if !ok {
		t.Fatalf("the done payload is %+v", h.payload(events.AgentDone))
	}

	spend, err := h.calls.SumByConversation(t.Context(), conversation)
	if err != nil {
		t.Fatalf("SumByConversation: %v", err)
	}
	if spend.Calls != finished.Calls || spend.Calls != 3 {
		t.Fatalf("the ledger counted %d calls and the turn reported %d", spend.Calls, finished.Calls)
	}
	if spend.Usage.Input != finished.InputTokens || spend.Usage.Output != finished.OutputTokens {
		t.Fatalf("the ledger holds %+v and the turn reported %+v", spend.Usage, finished)
	}
	if spend.Usage.CachedInput != finished.CachedInputTokens {
		t.Fatalf("the ledger cached %d and the turn reported %d", spend.Usage.CachedInput, finished.CachedInputTokens)
	}
	if math.Abs(spend.USD-finished.USD) > 1e-9 {
		t.Fatalf("the ledger holds %v and the turn reported %v", spend.USD, finished.USD)
	}
}

func TestEveryRoundOfATurnAnnouncesWhatItSpent(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "TOOL:pages_tree{}\nFAKE: one read and an answer")

	rounds := make([]events.AgentUsagePayload, 0, 2)
	for _, event := range h.bus.Events() {
		if spent, ok := event.Payload.(events.AgentUsagePayload); ok {
			rounds = append(rounds, spent)
		}
	}
	if len(rounds) != 2 {
		t.Fatalf("the bus saw %d usage events for a turn of two model calls: %+v", len(rounds), rounds)
	}
	for index, spent := range rounds {
		if spent.Round != index+1 || spent.ConversationID != conversation {
			t.Fatalf("round %d reads %+v", index+1, spent)
		}
		if spent.Provider != "openai" || spent.Model != "chat" || spent.MessageID == "" {
			t.Fatalf("round %d reads %+v", index+1, spent)
		}
	}

	finished, ok := h.payload(events.AgentDone).(events.AgentDonePayload)
	if !ok || finished.Calls != 2 {
		t.Fatalf("the done payload is %+v", h.payload(events.AgentDone))
	}

	counted := 0
	for _, spent := range rounds {
		counted += spent.InputTokens
	}
	if counted != finished.InputTokens {
		t.Fatalf("the rounds counted %d input tokens and the turn reported %d", counted, finished.InputTokens)
	}
}

func TestTheConversationHistoryIsReplayedOnTheNextTurn(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "FAKE: first")

	body, version, err := sqlite.NewConversationHistoryRepo(h.store).Load(t.Context(), conversation)
	if err != nil || version == 0 {
		t.Fatalf("Load = %s, %d, %v", body, version, err)
	}

	var history map[string]any
	if unmarshalErr := json.Unmarshal(body, &history); unmarshalErr != nil {
		t.Fatalf("decode the stored history: %v", unmarshalErr)
	}
	messages, ok := history["messages"].([]any)
	if !ok || len(messages) == 0 {
		t.Fatalf("the stored history is %s", body)
	}
}

func TestTheStoredHistoryReplaysAShorterToolResultThanTheTurnSaw(t *testing.T) {
	t.Parallel()

	const ceiling = 4096

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	for i := range 40 {
		sqlitetest.Page(t, store, owner.ID, fmt.Sprintf("/coffee/espresso/single-origin-blend-%02d/", i))
	}

	h := buildTuned(t, store, fake.NewGollem(), &applicationtest.Recorder{}, func(deps *agentapp.Deps) {
		deps.MaxToolResult = func() int { return ceiling }
	})
	h.siteID = owner.ID

	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "TOOL:pages_list{}\nFAKE: forty pages")

	outcome, ok := h.payload(events.AgentToolFinished).(events.AgentToolFinishedPayload)
	if !ok || outcome.Tool != "pages_list" || len(outcome.Result) == 0 {
		t.Fatalf("the finished payload is %+v", h.payload(events.AgentToolFinished))
	}

	body, _, err := sqlite.NewConversationHistoryRepo(store).Load(t.Context(), conversation)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	var history gollem.History
	if unmarshalErr := json.Unmarshal(body, &history); unmarshalErr != nil {
		t.Fatalf("decode the stored history: %v", unmarshalErr)
	}

	replayed := 0
	for _, message := range history.Messages {
		for _, content := range message.Contents {
			if content.Type != gollem.MessageContentTypeToolResponse {
				continue
			}
			replayed++
			if len(content.Data) > agentapp.HistoryToolResultBytes(ceiling)+historyFenceSlack {
				t.Fatalf("the stored result is %d bytes, over the history ceiling", len(content.Data))
			}
			if len(content.Data) >= len(outcome.Result) {
				t.Fatalf("the stored result is %d bytes of the %d the turn saw",
					len(content.Data), len(outcome.Result))
			}
			if _, decodeErr := content.GetToolResponseContent(); decodeErr != nil {
				t.Fatalf("the stored result no longer decodes: %v", decodeErr)
			}
		}
	}
	if replayed != 1 {
		t.Fatalf("the stored history holds %d tool results, want 1", replayed)
	}
}

func TestTheStoredHistoryHoldsTheAnswerOnce(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "FAKE: Отличная идея.")

	body, _, err := sqlite.NewConversationHistoryRepo(h.store).Load(t.Context(), conversation)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	var history gollem.History
	if unmarshalErr := json.Unmarshal(body, &history); unmarshalErr != nil {
		t.Fatalf("decode the stored history: %v", unmarshalErr)
	}

	spoken := 0
	for _, message := range history.Messages {
		if message.Role == gollem.RoleAssistant {
			spoken++
		}
	}
	if spoken != 1 {
		t.Fatalf("the stored history holds %d assistant messages, want 1: %s", spoken, body)
	}
}

func TestASavedToolRowCarriesTheStatusTheLedgerRecorded(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{
		Mode: string(domainagent.ModeAutonomous),
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	conversation := created.Conversation.ID

	h.send(t, conversation, "TOOL:pages_tree{}\nFAKE: no site here")

	live, ok := h.payload(events.AgentToolFinished).(events.AgentToolFinishedPayload)
	if !ok || live.Status != string(domainagent.CallDenied) {
		t.Fatalf("the live row is %+v", h.payload(events.AgentToolFinished))
	}

	saved := h.messages(t, conversation)
	rows := 0
	for _, message := range saved {
		if message.Role != "tool" {
			continue
		}
		rows++
		if message.ToolStatus != string(domainagent.CallDenied) {
			t.Fatalf("the saved row reads %q, want the %q the ledger holds",
				message.ToolStatus, domainagent.CallDenied)
		}
	}
	if rows != 1 {
		t.Fatalf("the transcript holds %d tool rows", rows)
	}
}

func TestASavedToolRowThatSucceededReadsAsDone(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "TOOL:pages_tree{}\nFAKE: the tree is empty")

	for _, message := range h.messages(t, conversation) {
		if message.Role == "tool" && message.ToolStatus != string(domainagent.CallOK) {
			t.Fatalf("the saved row reads %q, want ok", message.ToolStatus)
		}
	}
}

func TestSendRefusesWhatItCannotAnswer(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)

	if _, err := h.service.Send(t.Context(), agentapp.SendRequest{ConversationID: conversation, Text: "  "}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Send of nothing = %v", err)
	}
	if _, err := h.service.Send(t.Context(), agentapp.SendRequest{Text: "hello"}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Send without a conversation = %v", err)
	}
}

func TestSetModeAndListingsFollowTheConversation(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeConfirm)

	changed, err := h.service.SetMode(t.Context(), agentapp.SetModeRequest{
		ConversationID: conversation, Mode: string(domainagent.ModeAutonomous),
	})
	if err != nil || changed.Conversation.Mode != string(domainagent.ModeAutonomous) {
		t.Fatalf("SetMode = %+v, %v", changed, err)
	}
	if _, err = h.service.SetMode(t.Context(), agentapp.SetModeRequest{
		ConversationID: conversation, Mode: "silent",
	}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("SetMode to an unknown mode = %v", err)
	}

	listed, err := h.service.ListConversations(t.Context(), agentapp.ListConversationsRequest{SiteID: h.siteID})
	if err != nil || len(listed.Items) != 1 {
		t.Fatalf("ListConversations = %+v, %v", listed, err)
	}

	cancelled, err := h.service.Cancel(t.Context(), agentapp.CancelRequest{ConversationID: conversation})
	if err != nil || cancelled.Cancelled {
		t.Fatalf("Cancel of an idle conversation = %+v, %v", cancelled, err)
	}
}

func TestCreateConversationChecksItsSite(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	global, err := h.service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{})
	if err != nil || global.Conversation.SiteID != nil || global.Conversation.Mode != "confirm" {
		t.Fatalf("CreateConversation = %+v, %v", global, err)
	}

	if _, err = h.service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{
		SiteID: "6fb0b1d6-1ad5-4a26-8f26-2f2b6d9e4b23",
	}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("CreateConversation on an unknown site = %v", err)
	}
	if _, err = h.service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{
		SiteID: h.siteID, Mode: "silent",
	}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("CreateConversation in an unknown mode = %v", err)
	}
}

func TestAConversationWithoutASiteStillAnswers(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	created, err := h.service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{
		Mode: string(domainagent.ModeAutonomous),
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	h.send(t, created.Conversation.ID, "TOOL:pages_tree{}\nFAKE: no site here")

	outcome, ok := h.payload(events.AgentToolFinished).(events.AgentToolFinishedPayload)
	if !ok || outcome.Status != string(domainagent.CallDenied) {
		t.Fatalf("a site scoped tool in a siteless conversation = %+v", h.payload(events.AgentToolFinished))
	}
}

func TestTheListingsRefuseWhatTheyCannotRead(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	if _, err := h.service.ListMessages(t.Context(), agentapp.ListMessagesRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("ListMessages without a conversation = %v", err)
	}
	if _, err := h.service.ListPendingActions(t.Context(), agentapp.ListPendingActionsRequest{
		Status: "later",
	}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("ListPendingActions of an unknown status = %v", err)
	}
	if _, err := h.service.Cancel(t.Context(), agentapp.CancelRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Cancel without a conversation = %v", err)
	}
	if _, err := h.service.SetMode(t.Context(), agentapp.SetModeRequest{Mode: "confirm"}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("SetMode without a conversation = %v", err)
	}
}

func TestConfirmRefusesWhatItCannotResolve(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	if _, err := h.service.Confirm(t.Context(), agentapp.ConfirmRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Confirm without an action = %v", err)
	}
	if _, err := h.service.Confirm(t.Context(), agentapp.ConfirmRequest{
		ActionID: "6fb0b1d6-1ad5-4a26-8f26-2f2b6d9e4b23",
	}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Confirm of an unknown action = %v", err)
	}
}

func TestAFailedToolIsRecordedAsFailedOnTheAction(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeConfirm)
	h.send(t, conversation, `TOOL:pages_delete{"id":"6fb0b1d6-1ad5-4a26-8f26-2f2b6d9e4b23"}`+"\nFAKE: waiting")

	pending, err := h.service.ListPendingActions(t.Context(), agentapp.ListPendingActionsRequest{
		ConversationID: conversation,
	})
	if err != nil || len(pending.Items) != 1 {
		t.Fatalf("the pending actions are %+v, %v", pending, err)
	}

	settled, err := h.service.Confirm(t.Context(), agentapp.ConfirmRequest{
		ActionID: pending.Items[0].ID, Approve: true,
	})
	if err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if settled.Action.Status != string(domainagent.ActionFailed) || settled.Action.Error == "" {
		t.Fatalf("the action settled as %+v", settled.Action)
	}
}
