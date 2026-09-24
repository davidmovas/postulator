package agent_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/settings"
)

type blockingRunner struct {
	release chan struct{}
	entered chan struct{}
}

func (b *blockingRunner) Run(context.Context, agentapp.RunSpec) (agentapp.RunResult, error) {
	close(b.entered)
	<-b.release
	return agentapp.RunResult{Text: "done"}, nil
}

type failingBus struct{}

func (failingBus) Publish(events.Type, any) error {
	return errors.New(errors.External, "no window is listening")
}

func blocked(t *testing.T) (*agentapp.Service, string, *blockingRunner) {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	now := clock.NewFake(sqlitetest.Stamp)
	runner := &blockingRunner{release: make(chan struct{}), entered: make(chan struct{})}
	siteRepo := sqlite.NewSiteRepo(store)
	pageRepo := sqlite.NewPageRepo(store)
	templateService := templates.New(sqlite.NewTemplateRepo(store), sqlite.NewLinkPolicyRepo(store),
		pageRepo, sqlite.NewEntityRepo(store), siteRepo, store, &applicationtest.Recorder{}, now)

	service := agentapp.New(agentapp.Deps{
		Conversations: sqlite.NewConversationRepo(store),
		Messages:      sqlite.NewMessageRepo(store),
		Actions:       sqlite.NewPendingActionRepo(store),
		Calls:         sqlite.NewToolCallRepo(store),
		Sites:         siteRepo,
		Reports: reports.New(reports.Deps{
			Entities: sqlite.NewEntityRepo(store), Edges: sqlite.NewEdgeRepo(store), Pages: pageRepo,
			Links: sqlite.NewPageLinkRepo(store), Runs: sqlite.NewRunRepo(store), Items: sqlite.NewRunItemRepo(store),
			Artifacts: sqlite.NewArtifactRepo(store), Sites: siteRepo, Specs: templateService, Policies: templateService,
		}),
		Templates: templateService,
		Profiles:  chatProfiles{},
		LLM:       &scriptedTitler{text: "a name the blocked turn never reaches"},
		Registry:  tools.New(tools.Deps{Clock: now}),
		Runner:    runner,
		Publisher: failingBus{},
		Clock:     now,
	})
	t.Cleanup(func() {
		close(runner.release)
		service.Close()
	})

	created, err := service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{
		SiteID: owner.ID, Mode: string(domainagent.ModeAutonomous),
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	return service, created.Conversation.ID, runner
}

func TestASecondMessageWaitsForTheTurnInFlight(t *testing.T) {
	t.Parallel()

	service, conversation, runner := blocked(t)

	if _, err := service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "the first question",
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	<-runner.entered

	if _, err := service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversation, Text: "the second question",
	}); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("a second Send = %v, want a conflict", err)
	}

	cancelled, err := service.Cancel(t.Context(), agentapp.CancelRequest{ConversationID: conversation})
	if err != nil || !cancelled.Cancelled {
		t.Fatalf("Cancel = %+v, %v", cancelled, err)
	}

	listed, err := service.ListMessages(t.Context(), agentapp.ListMessagesRequest{ConversationID: conversation})
	if err != nil || len(listed.Items) != 1 {
		t.Fatalf("the transcript is %+v, %v", listed.Items, err)
	}
}

func TestADroppedEventIsNoted(t *testing.T) {
	t.Parallel()

	turns := agentapp.NewTurns(nil)
	turns.Note(nil)
	if len(turns.Dropped()) != 0 {
		t.Fatal("nothing to note is noted as nothing")
	}

	turns.Note(errors.New(errors.External, "no window is listening"))
	if dropped := turns.Dropped(); len(dropped) != 1 {
		t.Fatalf("the dropped events are %v", dropped)
	}
	turns.Close()
}

func TestAnActionThatMovedOnIsAConflict(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	conversation := h.conversation(t, domainagent.ModeConfirm)

	actions := sqlite.NewPendingActionRepo(h.store)
	action, err := domainagent.NewPendingAction(domainagent.PendingAction{
		ID: id.New(), ConversationID: conversation, Tool: "pages_delete", Summary: "delete a page",
		Args: json.RawMessage(`{"id":"p1"}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewPendingAction: %v", err)
	}
	if insertErr := actions.Insert(t.Context(), action); insertErr != nil {
		t.Fatalf("Insert: %v", insertErr)
	}

	claimed, err := actions.Transition(t.Context(), action.ID, domainagent.ActionPending,
		domainagent.ActionApproved, nil, "", sqlitetest.Stamp.Add(time.Minute))
	if err != nil || !claimed {
		t.Fatalf("Transition = %t, %v", claimed, err)
	}

	if _, err = h.service.Confirm(t.Context(), agentapp.ConfirmRequest{
		ActionID: action.ID, Approve: true,
	}); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Confirm of an action another resolver claimed = %v", err)
	}
	if _, err = h.service.Confirm(t.Context(), agentapp.ConfirmRequest{ActionID: action.ID}); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("a rejection of a claimed action = %v", err)
	}
}

func TestTheAgentSettingsCarryTheirDefaults(t *testing.T) {
	t.Parallel()

	values := settings.Default().NewValues()
	if agentapp.LoopLimit(values) != agentapp.DefaultLoopLimit ||
		agentapp.HistoryBudgetChars(values) != agentapp.DefaultHistoryBudgetChars ||
		agentapp.MaxToolResultBytes(values) != agentapp.DefaultMaxToolResultBytes {
		t.Fatal("the agent settings do not carry their defaults")
	}

	unknown, err := settings.Default().Apply(values, map[string]json.RawMessage{
		"agent.loopLimit":          json.RawMessage(`4`),
		"agent.historyBudgetChars": json.RawMessage(`9000`),
		"agent.maxToolResultBytes": json.RawMessage(`2048`),
	})
	if err != nil || len(unknown) != 0 {
		t.Fatalf("Apply = %v, %v", unknown, err)
	}
	if agentapp.LoopLimit(values) != 4 || agentapp.HistoryBudgetChars(values) != 9000 ||
		agentapp.MaxToolResultBytes(values) != 2048 {
		t.Fatal("the agent settings did not take their stored values")
	}
}
