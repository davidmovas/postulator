package agent_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/gollem-dev/gollem"
	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	agentrunner "github.com/davidmovas/postulator/internal/transport/agent"
)

const (
	pollInterval = 5 * time.Millisecond
	pollTimeout  = 10 * time.Second
)

type staticFactory struct {
	client gollem.LLMClient
	err    error
}

func (f staticFactory) New(context.Context, domainllm.ModelRef) (gollem.LLMClient, error) {
	return f.client, f.err
}

type chatProfiles struct {
	err error
}

func (p chatProfiles) Resolve(context.Context, string, domainllm.Role, map[domainllm.Role]domainllm.ModelRef) (domainllm.ModelRef, error) {
	if p.err != nil {
		return domainllm.ModelRef{}, p.err
	}
	return domainllm.ModelRef{Provider: "openai", Model: "chat"}, nil
}

type fixedCatalog struct{}

func (fixedCatalog) Lookup(context.Context, domainllm.ModelRef) (domainllm.ModelInfo, error) {
	return domainllm.ModelInfo{InputUSDPerM: 1, OutputUSDPerM: 2}, nil
}

type harness struct {
	store    *sqlite.Store
	model    *fake.Gollem
	service  *agentapp.Service
	bus      *applicationtest.Recorder
	registry *tools.Registry
	calls    *sqlite.LLMCallRepo
	pages    *sqlite.PageRepo
	siteID   string
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	model := fake.NewGollem()
	bus := &applicationtest.Recorder{}

	built := build(t, store, model, bus)
	built.siteID = owner.ID
	return built
}

func build(t *testing.T, store *sqlite.Store, model *fake.Gollem, bus *applicationtest.Recorder) *harness {
	t.Helper()

	now := clock.NewFake(sqlitetest.Stamp)
	siteRepo := sqlite.NewSiteRepo(store)
	pageRepo := sqlite.NewPageRepo(store)
	entityRepo := sqlite.NewEntityRepo(store)
	edgeRepo := sqlite.NewEdgeRepo(store)
	linkRepo := sqlite.NewPageLinkRepo(store)
	actionRepo := sqlite.NewPendingActionRepo(store)
	callRepo := sqlite.NewLLMCallRepo(store)

	templateService := templates.New(sqlite.NewTemplateRepo(store), sqlite.NewLinkPolicyRepo(store),
		pageRepo, siteRepo, store, bus, now)
	registered := tools.New(tools.Deps{
		Sites:     sites.New(siteRepo, nil, store, now),
		Pages:     pages.New(pageRepo, linkRepo, entityRepo, siteRepo, store, bus, now),
		Templates: templateService,
		Reports: reports.New(entityRepo, edgeRepo, pageRepo, linkRepo, sqlite.NewRunRepo(store),
			sqlite.NewRunItemRepo(store), sqlite.NewArtifactRepo(store)),
		Actions:   actionRepo,
		Publisher: bus,
		Clock:     now,
	})

	runner := agentrunner.New(agentrunner.Deps{
		Factory:  staticFactory{client: model},
		Registry: registered,
		History:  sqlite.NewConversationHistoryRepo(store),
		Calls:    callRepo,
		Catalog:  fixedCatalog{},
		Clock:    now,
		Logger:   zaptest.NewLogger(t),
	}, agentrunner.Config{})

	service := agentapp.New(agentapp.Deps{
		Conversations: sqlite.NewConversationRepo(store),
		Messages:      sqlite.NewMessageRepo(store),
		Actions:       actionRepo,
		Calls:         sqlite.NewToolCallRepo(store),
		Sites:         siteRepo,
		Reports: reports.New(entityRepo, edgeRepo, pageRepo, linkRepo, sqlite.NewRunRepo(store),
			sqlite.NewRunItemRepo(store), sqlite.NewArtifactRepo(store)),
		Templates: templateService,
		Profiles:  chatProfiles{},
		Registry:  registered,
		Runner:    runner,
		Publisher: bus,
		Clock:     now,
	})
	t.Cleanup(service.Close)

	return &harness{
		store: store, model: model, service: service, bus: bus, registry: registered,
		calls: callRepo, pages: pageRepo,
	}
}

func (h *harness) conversation(t *testing.T, mode domainagent.Mode) string {
	t.Helper()

	created, err := h.service.CreateConversation(t.Context(), agentapp.CreateConversationRequest{
		SiteID: h.siteID, Title: "plan the hub", Mode: string(mode),
	})
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	return created.Conversation.ID
}

func (h *harness) send(t *testing.T, conversationID, text string) {
	t.Helper()

	if _, err := h.service.Send(t.Context(), agentapp.SendRequest{
		ConversationID: conversationID, Text: text,
	}); err != nil {
		t.Fatalf("Send: %v", err)
	}
	h.settled(t)
}

func (h *harness) settled(t *testing.T) {
	t.Helper()

	waitFor(t, "the turn to finish", func() bool {
		for _, event := range h.bus.Events() {
			if event.Type == events.AgentDone {
				return true
			}
		}
		return false
	})
}

func (h *harness) seen() []events.Type {
	recorded := h.bus.Events()
	out := make([]events.Type, 0, len(recorded))
	for _, event := range recorded {
		out = append(out, event.Type)
	}
	return out
}

func (h *harness) payload(eventType events.Type) any {
	for _, event := range h.bus.Events() {
		if event.Type == eventType {
			return event.Payload
		}
	}
	return nil
}

func (h *harness) messages(t *testing.T, conversationID string) []agentapp.Message {
	t.Helper()

	listed, err := h.service.ListMessages(t.Context(), agentapp.ListMessagesRequest{ConversationID: conversationID})
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	return listed.Items
}

func waitFor(t *testing.T, what string, done func() bool) {
	t.Helper()

	deadline := time.Now().Add(pollTimeout)
	for time.Now().Before(deadline) {
		if done() {
			return
		}
		time.Sleep(pollInterval)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func decode(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()

	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
	return out
}
