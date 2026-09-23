package agent_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/errors"
)

type scriptedRunner struct {
	before  func()
	release <-chan struct{}
	result  agentapp.RunResult
	err     error
	panics  bool
	waits   bool
}

func (r scriptedRunner) Run(ctx context.Context, spec agentapp.RunSpec) (agentapp.RunResult, error) {
	if r.before != nil {
		r.before()
	}
	if r.release != nil {
		<-r.release
	}
	if r.waits {
		<-ctx.Done()
		return agentapp.RunResult{}, errors.Wrap(ctx.Err(), errors.Cancelled, "the agent turn was stopped")
	}
	if r.panics {
		panic("the model client dereferenced a nil session")
	}
	if spec.Stream != nil && r.result.Text != "" {
		if err := spec.Stream.Delta(ctx, 7, r.result.Text); err != nil {
			return agentapp.RunResult{}, err
		}
	}
	return r.result, r.err
}

func scripted(t *testing.T, runner agentapp.Runner, timeout func() time.Duration) *harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	bus := &applicationtest.Recorder{}

	built := buildTuned(t, store, fake.NewGollem(), bus, func(deps *agentapp.Deps) {
		deps.Runner = runner
		deps.TurnTimeout = timeout
	})
	built.siteID = owner.ID
	return built
}

func doneOf(t *testing.T, h *harness) events.AgentDonePayload {
	t.Helper()

	h.settled(t)
	finished, ok := h.payload(events.AgentDone).(events.AgentDonePayload)
	if !ok {
		t.Fatalf("the done payload is %+v", h.payload(events.AgentDone))
	}
	return finished
}

func TestSendAnswersWithTheAssistantIdEveryEventCarries(t *testing.T) {
	t.Parallel()

	h := scripted(t, scriptedRunner{result: agentapp.RunResult{Text: "the graph holds eleven hubs"}}, nil)
	conversation := h.conversation(t, domainagent.ModeAutonomous)

	sent, err := h.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "how many hubs?",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}
	if sent.MessageID == "" || sent.AssistantMessageID == "" {
		t.Fatalf("Send answered %+v, want both ids", sent)
	}
	if sent.MessageID == sent.AssistantMessageID {
		t.Fatal("the user message and the assistant message share an id")
	}

	finished := doneOf(t, h)
	if finished.MessageID != sent.AssistantMessageID {
		t.Errorf("agent.done carries %q, want %q", finished.MessageID, sent.AssistantMessageID)
	}
	if finished.Code != "" {
		t.Errorf("a turn that answered carries code %q, want none", finished.Code)
	}

	delta, ok := h.payload(events.AgentDelta).(events.AgentDeltaPayload)
	if !ok || delta.MessageID != sent.AssistantMessageID {
		t.Errorf("agent.delta carries %+v, want the assistant id %q", h.payload(events.AgentDelta), sent.AssistantMessageID)
	}
}

func TestATurnThatEndsBeforeSendReturnsIsNotRunning(t *testing.T) {
	t.Parallel()

	h := scripted(t, scriptedRunner{err: errors.New(errors.Unauthorized, "no key is stored for openai")}, nil)
	conversation := h.conversation(t, domainagent.ModeAutonomous)

	sent, err := h.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "who am I talking to?",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	finished := doneOf(t, h)
	if finished.Code != string(errors.Unauthorized) {
		t.Errorf("code = %q, want %q", finished.Code, errors.Unauthorized)
	}
	if finished.MessageID != sent.AssistantMessageID {
		t.Errorf("agent.done carries %q, want %q", finished.MessageID, sent.AssistantMessageID)
	}

	status, err := h.service.Status(t.Context(), agentapp.StatusRequest{ConversationID: conversation})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if status.Running || status.MessageID != "" || status.LastSeq != 0 || status.StartedAt.String() != "" {
		t.Fatalf("Status = %+v, want a finished turn", status)
	}
}

func TestStatusBeforeDuringAndAfterATurn(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})
	h := scripted(t, scriptedRunner{release: release, result: agentapp.RunResult{Text: "eleven"}}, nil)
	conversation := h.conversation(t, domainagent.ModeAutonomous)

	before, err := h.service.Status(t.Context(), agentapp.StatusRequest{ConversationID: conversation})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if before.Running {
		t.Fatalf("Status before the turn = %+v", before)
	}

	sent, err := h.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "how many hubs?",
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	during, err := h.service.Status(t.Context(), agentapp.StatusRequest{ConversationID: conversation})
	if err != nil {
		t.Fatalf("Status: %v", err)
	}
	if !during.Running {
		t.Fatalf("Status during the turn = %+v, want it running", during)
	}
	if during.MessageID != sent.AssistantMessageID {
		t.Errorf("Status names %q, want %q", during.MessageID, sent.AssistantMessageID)
	}
	if during.StartedAt.String() == "" {
		t.Error("a running turn has no start instant")
	}

	close(release)
	h.settled(t)

	waitFor(t, "the registry to forget the turn", func() bool {
		after, statusErr := h.service.Status(t.Context(), agentapp.StatusRequest{ConversationID: conversation})
		return statusErr == nil && !after.Running
	})
}

func TestStatusReportsTheLastStreamedSeq(t *testing.T) {
	t.Parallel()

	observed := make(chan agentapp.StatusResponse, 1)
	release := make(chan struct{})

	var built *harness
	runner := scriptedRunner{release: release, result: agentapp.RunResult{Text: "the graph holds eleven hubs"}}
	built = scripted(t, streamingRunner{inner: runner, after: func(conversationID string) {
		status, err := built.service.Status(context.Background(), agentapp.StatusRequest{ConversationID: conversationID})
		if err == nil {
			observed <- status
		}
	}}, nil)

	conversation := built.conversation(t, domainagent.ModeAutonomous)
	if _, err := built.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "how many hubs?",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	close(release)

	select {
	case status := <-observed:
		if status.LastSeq != 7 {
			t.Fatalf("LastSeq = %d, want the seq the stream carried", status.LastSeq)
		}
	case <-time.After(pollTimeout):
		t.Fatal("the runner never streamed a delta")
	}
	built.settled(t)
}

func TestATerminalEventCarriesItsCode(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		err     error
		code    string
		message string
	}{
		{
			name: "a missing key", err: errors.New(errors.Unauthorized, "no key is stored for openai"),
			code: string(errors.Unauthorized), message: "no key is stored for openai",
		},
		{
			name: "a stopped turn", err: errors.New(errors.Cancelled, "the agent turn was stopped"),
			code: string(errors.Cancelled), message: "the agent turn was stopped",
		},
		{
			name: "a provider that broke", err: errors.New(errors.External, "the model could not answer"),
			code: string(errors.External), message: "the model could not answer",
		},
		{
			name: "a failure that is not ours", err: context.DeadlineExceeded,
			code: string(errors.Internal), message: "unexpected internal error",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := scripted(t, scriptedRunner{err: tc.err}, nil)
			conversation := h.conversation(t, domainagent.ModeAutonomous)
			if _, err := h.service.Send(t.Context(), agentapp.SendRequest{
				ConversationID: conversation, Text: "go on",
			}); err != nil {
				t.Fatalf("Send: %v", err)
			}

			finished := doneOf(t, h)
			if finished.Code != tc.code {
				t.Errorf("code = %q, want %q", finished.Code, tc.code)
			}
			if finished.Error != tc.message {
				t.Errorf("error = %q, want %q", finished.Error, tc.message)
			}
		})
	}
}

func TestAPanickingTurnEndsInternallyAndTheProcessSurvives(t *testing.T) {
	t.Parallel()

	h := scripted(t, scriptedRunner{panics: true}, nil)
	conversation := h.conversation(t, domainagent.ModeAutonomous)

	if _, err := h.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "break something",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	finished := doneOf(t, h)
	if finished.Code != string(errors.Internal) {
		t.Errorf("code = %q, want %q", finished.Code, errors.Internal)
	}
	if finished.Error != "the agent turn failed unexpectedly" {
		t.Errorf("error = %q", finished.Error)
	}

	second, err := h.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "and again",
	})
	if err != nil {
		t.Fatalf("the service did not survive the panic: %v", err)
	}
	if second.AssistantMessageID == "" {
		t.Fatal("the second turn minted no assistant id")
	}
	h.settled(t)
}

func TestADeadlineEndsTheTurnWithItsOwnMessage(t *testing.T) {
	t.Parallel()

	h := scripted(t, scriptedRunner{waits: true}, func() time.Duration { return 30 * time.Millisecond })
	conversation := h.conversation(t, domainagent.ModeAutonomous)

	if _, err := h.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "take your time",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	finished := doneOf(t, h)
	if finished.Code != string(errors.Cancelled) {
		t.Errorf("code = %q, want %q", finished.Code, errors.Cancelled)
	}
	if finished.Error != "the agent turn ran out of time" {
		t.Errorf("error = %q, want the deadline message", finished.Error)
	}
}

func TestACancelledTurnKeepsTheStopMessage(t *testing.T) {
	t.Parallel()

	h := scripted(t, scriptedRunner{waits: true}, nil)
	conversation := h.conversation(t, domainagent.ModeAutonomous)

	if _, err := h.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "take your time",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}

	stopped, err := h.service.Cancel(t.Context(), agentapp.CancelRequest{ConversationID: conversation})
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if !stopped.Cancelled {
		t.Fatal("Cancel found no turn to stop")
	}

	finished := doneOf(t, h)
	if finished.Code != string(errors.Cancelled) {
		t.Errorf("code = %q, want %q", finished.Code, errors.Cancelled)
	}
	if finished.Error != "the agent turn was stopped" {
		t.Errorf("error = %q, want the stop message", finished.Error)
	}
}

func TestStatusNeedsAConversation(t *testing.T) {
	t.Parallel()

	h := scripted(t, scriptedRunner{}, nil)
	if _, err := h.service.Status(t.Context(), agentapp.StatusRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Status without a conversation = %v, want INVALID", err)
	}
}

type streamingRunner struct {
	inner scriptedRunner
	after func(conversationID string)
}

func (r streamingRunner) Run(ctx context.Context, spec agentapp.RunSpec) (agentapp.RunResult, error) {
	result, err := r.inner.Run(ctx, spec)
	if r.after != nil {
		r.after(spec.Binding.ConversationID)
	}
	return result, err
}

func TestADirectiveFailsATurnWithTheCodeItNames(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		prompt string
		code   string
	}{
		{name: "FAIL names a code", prompt: "FAIL:UNAUTHORIZED", code: string(errors.Unauthorized)},
		{name: "ERROR names a code", prompt: "ERROR:EXTERNAL", code: string(errors.External)},
		{name: "FAIL under a sentence", prompt: "show me the error row\nFAIL:RATE_LIMITED", code: string(errors.RateLimited)},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			h := newHarness(t)
			conversation := h.conversation(t, domainagent.ModeAutonomous)
			if _, err := h.service.Send(t.Context(), agentapp.SendRequest{
				ConversationID: conversation, Text: tc.prompt,
			}); err != nil {
				t.Fatalf("Send: %v", err)
			}

			finished := doneOf(t, h)
			if finished.Code != tc.code {
				t.Fatalf("code = %q, want %q", finished.Code, tc.code)
			}
			if finished.Error == "" {
				t.Fatal("the terminal event carries no message")
			}
		})
	}
}

func TestTheTurnReadsItsLimitsOnEveryTurn(t *testing.T) {
	t.Parallel()

	var seen []agentapp.RunSpec
	var mu sync.Mutex

	limit, budget, ceiling := 3, 4000, 2048
	recorded := recordingRunner{observe: func(spec agentapp.RunSpec) {
		mu.Lock()
		defer mu.Unlock()
		seen = append(seen, spec)
	}}

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	bus := &applicationtest.Recorder{}
	h := buildTuned(t, store, fake.NewGollem(), bus, func(deps *agentapp.Deps) {
		deps.Runner = recorded
		deps.LoopLimit = func() int { return limit }
		deps.HistoryBudget = func() int { return budget }
		deps.MaxToolResult = func() int { return ceiling }
	})
	h.siteID = owner.ID

	conversation := h.conversation(t, domainagent.ModeAutonomous)
	h.send(t, conversation, "first turn")

	limit, budget, ceiling = 9, 9000, 4096
	bus.Reset()
	h.send(t, conversation, "second turn")

	mu.Lock()
	defer mu.Unlock()

	if len(seen) != 2 {
		t.Fatalf("the runner saw %d turns, want 2", len(seen))
	}
	if seen[0].LoopLimit != 3 || seen[0].HistoryBudget != 4000 || seen[0].MaxToolResult != 2048 {
		t.Fatalf("the first turn ran with %+v", seen[0])
	}
	if seen[1].LoopLimit != 9 || seen[1].HistoryBudget != 9000 || seen[1].MaxToolResult != 4096 {
		t.Fatalf("the second turn kept the old limits: %+v", seen[1])
	}
}

type recordingRunner struct {
	observe func(agentapp.RunSpec)
}

func (r recordingRunner) Run(_ context.Context, spec agentapp.RunSpec) (agentapp.RunResult, error) {
	r.observe(spec)
	return agentapp.RunResult{Text: "done"}, nil
}
