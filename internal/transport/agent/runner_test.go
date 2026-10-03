package agent_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/llm/retry"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	llmport "github.com/davidmovas/postulator/internal/application/llm"
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

	if asked := h.model.Requests(); len(asked) != 2 {
		t.Fatalf("the model was asked %d times for one call and one answer", len(asked))
	}
	if !fencedForTheModel(h.model.Requests()) {
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

type refusingFirst struct {
	next     llmport.Client
	mu       sync.Mutex
	refusals int
	err      error
}

func (r *refusingFirst) refuse() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.refusals == 0 {
		return false
	}
	r.refusals--
	return true
}

func (r *refusingFirst) Complete(ctx context.Context, req llmport.Request) (llmport.Response, error) {
	if r.refuse() {
		return llmport.Response{}, r.err
	}
	return r.next.Complete(ctx, req)
}

func (r *refusingFirst) Stream(ctx context.Context, req llmport.Request) (<-chan llmport.Delta, error) {
	if r.refuse() {
		return nil, r.err
	}
	return r.next.Stream(ctx, req)
}

func TestAWaitIsToldToTheWindowAndEveryAttemptIsALedgerRow(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	model := fake.New()
	bus := &applicationtest.Recorder{}
	h := buildWired(t, store, model, wiring{
		provider: &refusingFirst{
			next: model, refusals: 1,
			err: errors.New(errors.RateLimited, "the model provider is rate limiting this key").WithRetry(3 * time.Millisecond),
		},
		outer: func(next llmport.Client) llmport.Client { return retry.New(next, 2, time.Millisecond) },
	}, bus, nil)
	h.siteID = owner.ID

	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "FAKE: answered after a wait")

	waited, ok := h.payload(events.AgentWaiting).(events.AgentWaitingPayload)
	if !ok {
		t.Fatalf("the window was never told about the wait: %v", h.seen())
	}
	if waited.Reason != string(errors.RateLimited) || waited.Attempt != 1 || waited.AfterMs != 3 {
		t.Fatalf("the wait reads %+v", waited)
	}
	if waited.ConversationID != conversation || waited.MessageID == "" {
		t.Fatalf("the wait names %+v", waited)
	}

	finished, ok := h.payload(events.AgentDone).(events.AgentDonePayload)
	if !ok || finished.Text != "answered after a wait" || finished.Calls != 1 {
		t.Fatalf("the done payload is %+v", h.payload(events.AgentDone))
	}

	listed, err := h.calls.List(t.Context(), domainllm.CallQuery{ConversationID: conversation}, paging.Request{Limit: 10})
	if err != nil || len(listed.Items) != 2 {
		t.Fatalf("the ledger holds %+v, %v; want the refused attempt and the answer", listed.Items, err)
	}
	statuses := []domainllm.CallStatus{listed.Items[0].Status, listed.Items[1].Status}
	if !slices.Contains(statuses, domainllm.CallError) || !slices.Contains(statuses, domainllm.CallOK) {
		t.Fatalf("the ledger rows read %v", statuses)
	}
}

type stallingAfter struct {
	next    llmport.Client
	mu      sync.Mutex
	rounds  int
	stalled chan struct{}
	once    sync.Once
}

func (s *stallingAfter) Complete(ctx context.Context, req llmport.Request) (llmport.Response, error) {
	return s.next.Complete(ctx, req)
}

func (s *stallingAfter) Stream(ctx context.Context, req llmport.Request) (<-chan llmport.Delta, error) {
	s.mu.Lock()
	s.rounds--
	stall := s.rounds < 0
	s.mu.Unlock()
	if !stall {
		return s.next.Stream(ctx, req)
	}

	s.once.Do(func() { close(s.stalled) })
	<-ctx.Done()
	return nil, errors.New(errors.Cancelled, "the model call was cancelled").WithInternal(ctx.Err())
}

func TestAStoppedTurnKeepsTheRoundsItFinished(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	model := fake.New()
	stalling := &stallingAfter{next: model, rounds: 1, stalled: make(chan struct{})}
	h := buildWired(t, store, model, wiring{provider: stalling}, &applicationtest.Recorder{}, nil)
	h.siteID = owner.ID

	conversation := h.conversation(t, domainagent.ModeAutonomous)
	if _, err := h.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "TOOL:pages_tree{}\nFAKE: never said",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	select {
	case <-stalling.stalled:
	case <-time.After(pollTimeout):
		t.Fatal("the second round never reached the model")
	}
	if stopped, err := h.service.Cancel(t.Context(), agentapp.CancelRequest{ConversationID: conversation}); err != nil || !stopped.Cancelled {
		t.Fatalf("Cancel = %+v, %v", stopped, err)
	}
	h.settled(t)

	finished, ok := h.payload(events.AgentDone).(events.AgentDonePayload)
	if !ok || finished.Code != string(errors.Cancelled) || finished.Error != "the agent turn was stopped" {
		t.Fatalf("the done payload is %+v", h.payload(events.AgentDone))
	}

	held, body := loadHistory(t, store, conversation)
	if len(held.Items) != 4 || held.Items[2].Call == nil || held.Items[3].Result == nil {
		t.Fatalf("the stored history is %s, want the finished round with its call and result", body)
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

type storedHistory struct {
	Format string            `json:"format"`
	Items  []llmport.Message `json:"items"`
}

func loadHistory(t *testing.T, store *sqlite.Store, conversation string) (held storedHistory, body []byte) {
	t.Helper()

	body, version, err := sqlite.NewConversationHistoryRepo(store).Load(t.Context(), conversation)
	if err != nil || version == 0 {
		t.Fatalf("Load = %s, %d, %v", body, version, err)
	}

	if unmarshalErr := json.Unmarshal(body, &held); unmarshalErr != nil {
		t.Fatalf("decode the stored history: %v", unmarshalErr)
	}
	if held.Format != "responses/1" {
		t.Fatalf("the stored history carries the format %q: %s", held.Format, body)
	}
	return held, body
}

func TestTheConversationHistoryIsReplayedOnTheNextTurn(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "TOOL:pages_tree{}\nFAKE: first")

	held, body := loadHistory(t, h.store, conversation)
	roles := make([]llmport.Role, 0, len(held.Items))
	for _, item := range held.Items {
		roles = append(roles, item.Role)
	}
	want := []llmport.Role{
		llmport.RoleDeveloper, llmport.RoleUser, llmport.RoleAssistant, llmport.RoleTool, llmport.RoleAssistant,
	}
	if !slices.Equal(roles, want) {
		t.Fatalf("the stored history holds %v, want %v: %s", roles, want, body)
	}

	h.bus.Reset()
	h.send(t, conversation, "FAKE: second")

	asked := h.model.Requests()
	replayed := asked[len(asked)-1].Messages
	if len(replayed) != len(held.Items)+2 {
		t.Fatalf("the second turn sent %d messages, want the %d stored and its own two", len(replayed), len(held.Items))
	}
	if replayed[1].Text != "TOOL:pages_tree{}\nFAKE: first" || replayed[3].Result == nil {
		t.Fatalf("the second turn did not replay the first: %+v", replayed)
	}
	if replayed[len(replayed)-2].Role != llmport.RoleDeveloper || replayed[len(replayed)-1].Text != "FAKE: second" {
		t.Fatalf("the second turn does not open with its own site context and question: %+v", replayed)
	}
}

func TestAHistoryInAnotherFormatIsForgottenAndTheTranscriptKept(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "FAKE: before the switch")

	older := []byte(`{"version":3,"llType":"openai","messages":[{"role":"user","contents":[{"type":"text","data":{"text":"hi"}}]}]}`)
	if err := sqlite.NewConversationHistoryRepo(h.store).Save(t.Context(), conversation, older, 3, sqlitetest.Stamp); err != nil {
		t.Fatalf("Save: %v", err)
	}

	h.bus.Reset()
	h.send(t, conversation, "FAKE: after the switch")

	asked := h.model.Requests()
	if sent := asked[len(asked)-1].Messages; len(sent) != 2 {
		t.Fatalf("a history in gollem's shape reached the model: %+v", sent)
	}
	if held, _ := loadHistory(t, h.store, conversation); len(held.Items) != 3 {
		t.Fatalf("the rewritten history holds %+v", held.Items)
	}
	if transcript := h.messages(t, conversation); len(transcript) != 4 {
		t.Fatalf("the transcript lost a message: %+v", transcript)
	}
}

func TestAStoredCallNeverCarriesACredential(t *testing.T) {
	t.Parallel()

	const secret = "sk-live-NeverStoreNeverStore4242"

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	model := fake.New(fake.WithScript(func(string) fake.Turn {
		return fake.Turn{Tool: "pages_tree", Args: json.RawMessage(`{"token":"` + secret + `"}`), Text: "refused"}
	}))
	h := build(t, store, model, &applicationtest.Recorder{})
	h.siteID = owner.ID

	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "read the tree with my token")

	held, body := loadHistory(t, h.store, conversation)
	if strings.Contains(string(body), secret) {
		t.Fatalf("the stored history carries the credential: %s", body)
	}

	masked := false
	for _, item := range held.Items {
		if item.Call != nil && strings.Contains(string(item.Call.Args), `"token":"***"`) {
			masked = true
		}
	}
	if !masked {
		t.Fatalf("the stored call no longer says a token was there: %s", body)
	}
}

func TestTheStoredHistoryReplaysAShorterToolResultThanTheTurnSaw(t *testing.T) {
	t.Parallel()

	const (
		ceiling      = 4096
		replayedCeil = 1024
	)

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	for i := range 40 {
		sqlitetest.Page(t, store, owner.ID, fmt.Sprintf("/coffee/espresso/single-origin-blend-%02d/", i))
	}

	h := buildTuned(t, store, fake.New(), &applicationtest.Recorder{}, func(deps *agentapp.Deps) {
		deps.MaxToolResult = func() int { return ceiling }
		deps.HistoryToolResult = func() int { return replayedCeil }
	})
	h.siteID = owner.ID

	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "TOOL:pages_list{}\nFAKE: forty pages")

	outcome, ok := h.payload(events.AgentToolFinished).(events.AgentToolFinishedPayload)
	if !ok || outcome.Tool != "pages_list" || len(outcome.Result) == 0 {
		t.Fatalf("the finished payload is %+v", h.payload(events.AgentToolFinished))
	}

	held, _ := loadHistory(t, store, conversation)
	replayed := 0
	for _, item := range held.Items {
		if item.Result == nil {
			continue
		}
		replayed++
		stored := item.Result.Output
		if len(stored) > replayedCeil+historyFenceSlack {
			t.Fatalf("the stored result is %d bytes, over the history ceiling", len(stored))
		}
		if len(stored) >= len(outcome.Result) {
			t.Fatalf("the stored result is %d bytes of the %d the turn saw", len(stored), len(outcome.Result))
		}
		if fenced := decode(t, stored); fenced["untrustedContent"] != true {
			t.Fatalf("the stored result lost its fence: %s", stored)
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

	held, body := loadHistory(t, h.store, conversation)
	spoken := 0
	for _, item := range held.Items {
		if item.Role == llmport.RoleAssistant {
			spoken++
		}
	}
	if spoken != 1 {
		t.Fatalf("the stored history holds %d assistant messages, want 1: %s", spoken, body)
	}
}

func TestEveryRoundAsksInTheShapeTheCacheKeeps(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeConfirm)
	h.send(t, conversation, "TOOL:pages_tree{}\nFAKE: the tree")

	sqlitetest.Page(t, h.store, h.siteID, "/coffee/fresh/")
	h.bus.Reset()
	h.send(t, conversation, "FAKE: one more page")

	asked := h.model.Requests()
	if len(asked) != 3 {
		t.Fatalf("the model was asked %d times, want three rounds", len(asked))
	}

	names := h.registry.Names()
	for index, req := range asked {
		if req.System != asked[0].System || strings.Contains(req.System, " entities, ") {
			t.Fatalf("round %d carries other instructions or the site in them: %q", index+1, req.System)
		}
		if req.CacheKey != "chat:confirm" || req.Effort != domainllm.EffortNone || req.Tier != domainllm.TierDefault {
			t.Fatalf("round %d asks with %q, %q and %q", index+1, req.CacheKey, req.Effort, req.Tier)
		}
		if req.Meta.Step != agentrunner.ChatStep || req.Meta.Role != domainllm.RoleChat || req.Meta.ConversationID != conversation {
			t.Fatalf("round %d is booked as %+v", index+1, req.Meta)
		}
		if len(req.Tools) != len(names) {
			t.Fatalf("round %d offers %d tools of %d", index+1, len(req.Tools), len(names))
		}
		for i, tool := range req.Tools {
			if tool.Name != names[i] || tool.Schema == nil || tool.Description == "" {
				t.Fatalf("round %d offers %+v at %d, want %s in registry order", index+1, tool, i, names[i])
			}
		}
		if !reflect.DeepEqual(req.Tools, asked[0].Tools) {
			t.Fatalf("round %d offers other tool bytes than the first", index+1)
		}
	}

	opening := asked[0].Messages[0]
	latest := asked[2].Messages[len(asked[2].Messages)-2]
	if opening.Role != llmport.RoleDeveloper || latest.Role != llmport.RoleDeveloper {
		t.Fatalf("the turns open with %+v and %+v, want the site context as a developer message", opening, latest)
	}
	if !strings.Contains(opening.Text, "0 pages") || !strings.Contains(latest.Text, "1 pages") {
		t.Fatalf("the contexts read %q and %q, want the page counts of their own turns", opening.Text, latest.Text)
	}
	if opening.Text == latest.Text {
		t.Fatalf("the second turn kept the first turn's site context after a page was added: %q", latest.Text)
	}
	if asked[2].Messages[0].Text != opening.Text {
		t.Fatal("the second turn rewrote the first turn's context, which breaks the cached prefix")
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
