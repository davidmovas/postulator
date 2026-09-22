package agent_test

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/adapters/llm/fake"
	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	agentapp "github.com/davidmovas/postulator/internal/application/agent"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/tools"
	domainagent "github.com/davidmovas/postulator/internal/domain/agent"
	domainllm "github.com/davidmovas/postulator/internal/domain/llm"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	agentrunner "github.com/davidmovas/postulator/internal/transport/agent"
)

type recordingStream struct {
	mu       sync.Mutex
	deltas   []string
	outcomes []agentapp.ToolOutcome
	rounds   []agentapp.RoundUsage
	err      error
}

func (r *recordingStream) Delta(_ context.Context, _ int64, text string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.deltas = append(r.deltas, text)
	return r.err
}

func (r *recordingStream) Spent(_ context.Context, round agentapp.RoundUsage) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.rounds = append(r.rounds, round)
	return r.err
}

func (r *recordingStream) Waiting(context.Context, agentapp.Wait) error {
	return r.err
}

func (r *recordingStream) ToolStarted(context.Context, string, string, json.RawMessage) error {
	return r.err
}

func (r *recordingStream) ToolFinished(_ context.Context, outcome agentapp.ToolOutcome) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.outcomes = append(r.outcomes, outcome)
	return r.err
}

func (r *recordingStream) settled() []agentapp.ToolOutcome {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]agentapp.ToolOutcome(nil), r.outcomes...)
}

type bareRunner struct {
	runner *agentrunner.Runner
	model  *fake.Gollem
	siteID string
}

func newBareRunner(t *testing.T, resultCap int) *bareRunner {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")
	sqlitetest.Page(t, store, owner.ID, "/coffee/")
	sqlitetest.Page(t, store, owner.ID, "/coffee/espresso/")

	now := clock.NewFake(sqlitetest.Stamp)
	bus := &applicationtest.Recorder{}
	model := fake.NewGollem()

	registered := tools.New(tools.Deps{
		Pages: pages.New(sqlite.NewPageRepo(store), sqlite.NewPageLinkRepo(store), sqlite.NewEntityRepo(store),
			sqlite.NewSiteRepo(store), store, bus, now, stubPreview{}),
		Actions:   sqlite.NewPendingActionRepo(store),
		Publisher: bus,
		Clock:     now,
	})

	return &bareRunner{
		runner: agentrunner.New(agentrunner.Deps{
			Factory:  staticFactory{client: model},
			Registry: registered,
			Catalog:  fixedCatalog{},
			Clock:    now,
			Logger:   zaptest.NewLogger(t),
		}, agentrunner.Config{MaxToolResultBytes: resultCap}),
		model: model, siteID: owner.ID,
	}
}

func (b *bareRunner) spec(input string, allowed []string, stream agentapp.Stream) agentapp.RunSpec {
	return agentapp.RunSpec{
		Binding: tools.Binding{
			SiteID: b.siteID, ConversationID: "conversation-1", Mode: domainagent.ModeAutonomous,
		},
		Ref:       domainllm.ModelRef{Provider: "openai", Model: "chat"},
		Context:   agentapp.SiteContext{SiteName: "Shop", Mode: string(domainagent.ModeAutonomous)},
		Input:     input,
		MessageID: "message-1",
		Allowed:   allowed,
		Stream:    stream,
		LoopLimit: 4,
	}
}

func TestAToolOutsideTheAllowListIsDenied(t *testing.T) {
	t.Parallel()

	b := newBareRunner(t, 0)
	stream := &recordingStream{}

	if _, err := b.runner.Run(t.Context(), b.spec("TOOL:pages_tree{}\nFAKE: done",
		[]string{"sites_list"}, stream)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	settled := stream.settled()
	if len(settled) != 1 || settled[0].Status != string(domainagent.CallDenied) {
		t.Fatalf("the outcomes are %+v", settled)
	}
	if !strings.Contains(settled[0].Error, "not open to this conversation") {
		t.Fatalf("the denial reads %q", settled[0].Error)
	}
}

func TestAnOversizedToolResultIsCappedBeforeItReachesTheModel(t *testing.T) {
	t.Parallel()

	b := newBareRunner(t, 120)
	stream := &recordingStream{}

	if _, err := b.runner.Run(t.Context(), b.spec("TOOL:pages_tree{}\nFAKE: done", nil, stream)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	settled := stream.settled()
	if len(settled) != 1 {
		t.Fatalf("the outcomes are %+v", settled)
	}

	result := decode(t, settled[0].Result)
	if result["truncated"] != true || result["preview"] == nil {
		t.Fatalf("the capped result is %v", result)
	}
	if result["untrustedContent"] != nil {
		t.Fatal("the cap runs inside the fence, so the fence wraps what the model finally sees")
	}
}

func TestACallTheSchemaRefusesBecomesAFailedRowTheModelCanRead(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		script string
		reads  string
	}{
		{
			name:   "a value outside the enum",
			script: `TOOL:graph_list_entities{"kind":"beverage"}`,
			reads:  "value not in enum",
		},
		{
			name:   "a number over the maximum",
			script: `TOOL:graph_add_edge{"fromEntityId":"a","toEntityId":"b","kind":"related","weight":5}`,
			reads:  "number too large",
		},
		{
			name:   "a required field missing inside a list",
			script: `TOOL:graph_set_anchors{"entityId":"e1","anchors":[{"text":"espresso"}]}`,
			reads:  "required parameter missing",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			b := newBareRunner(t, 0)
			stream := &recordingStream{}

			if _, err := b.runner.Run(t.Context(), b.spec(tc.script+"\nFAKE: done", nil, stream)); err != nil {
				t.Fatalf("Run: %v", err)
			}

			settled := stream.settled()
			if len(settled) != 1 || settled[0].Status != string(domainagent.CallError) {
				t.Fatalf("the outcomes are %+v", settled)
			}
			if !strings.Contains(settled[0].Error, tc.reads) {
				t.Fatalf("the row reads %q, which never says %q", settled[0].Error, tc.reads)
			}
		})
	}
}

func TestASiteScopedToolInASiteLessConversationIsDenied(t *testing.T) {
	t.Parallel()

	b := newBareRunner(t, 0)
	stream := &recordingStream{}

	spec := b.spec("TOOL:pages_tree{}\nFAKE: done", nil, stream)
	spec.Binding.SiteID = ""

	if _, err := b.runner.Run(t.Context(), spec); err != nil {
		t.Fatalf("Run: %v", err)
	}

	settled := stream.settled()
	if len(settled) != 1 || settled[0].Status != string(domainagent.CallDenied) {
		t.Fatalf("the outcomes are %+v", settled)
	}
	if !strings.Contains(settled[0].Error, "this tool works inside one site") {
		t.Fatalf("the denial reads %q", settled[0].Error)
	}
}

func TestALoopLimitEndsTheTurnAsAnExhaustedBudget(t *testing.T) {
	t.Parallel()

	b := newBareRunner(t, 0)
	stream := &recordingStream{}
	script := strings.Repeat("TOOL:pages_tree{}\n", 8)

	_, err := b.runner.Run(t.Context(), b.spec(script+"FAKE: done", nil, stream))
	if !errors.IsCode(err, errors.BudgetExceeded) {
		t.Fatalf("a turn that ran past its loop limit = %v, want a budget refusal", err)
	}
	if len(stream.settled()) == 0 {
		t.Fatal("the rows the turn did run were never recorded")
	}
}

func TestACappedListingStillDecodesAndSaysWhatWasDropped(t *testing.T) {
	t.Parallel()

	b := newBareRunner(t, 760)
	stream := &recordingStream{}

	if _, err := b.runner.Run(t.Context(), b.spec("TOOL:pages_list{}\nFAKE: done", nil, stream)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	settled := stream.settled()
	if len(settled) != 1 {
		t.Fatalf("the outcomes are %+v", settled)
	}

	var decoded struct {
		Truncated bool           `json:"truncated"`
		Dropped   map[string]int `json:"droppedItems"`
		Result    struct {
			Items []map[string]any `json:"items"`
		} `json:"result"`
	}
	if err := json.Unmarshal(settled[0].Result, &decoded); err != nil {
		t.Fatalf("the capped result no longer decodes: %v (%s)", err, settled[0].Result)
	}
	if !decoded.Truncated {
		t.Fatalf("the result was not cut: %s", settled[0].Result)
	}
	if len(decoded.Result.Items) == 0 || decoded.Dropped["items"] == 0 {
		t.Fatalf("the capped listing is %s", settled[0].Result)
	}
}

func TestTheFenceWrapsTheModelCopyAndTheLedgerKeepsTheRaw(t *testing.T) {
	t.Parallel()

	b := newBareRunner(t, 0)
	stream := &recordingStream{}

	if _, err := b.runner.Run(t.Context(), b.spec("TOOL:pages_tree{}\nFAKE: done", nil, stream)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	settled := stream.settled()
	if len(settled) != 1 {
		t.Fatalf("the outcomes are %+v", settled)
	}
	if outcome := decode(t, settled[0].Result); outcome["untrustedContent"] != nil || outcome["roots"] == nil {
		t.Fatalf("the audited result is %v; the audit records what the tool answered", outcome)
	}

	history, err := b.model.Sessions()[0].History()
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
		t.Fatal("the model was handed a tool result that was not fenced as untrusted data")
	}
}

func TestARunNeedsAnInputAModelAndALoopLimit(t *testing.T) {
	t.Parallel()

	b := newBareRunner(t, 0)

	cases := []struct {
		name   string
		mutate func(*agentapp.RunSpec)
	}{
		{name: "no input", mutate: func(s *agentapp.RunSpec) { s.Input = "  " }},
		{name: "no loop limit", mutate: func(s *agentapp.RunSpec) { s.LoopLimit = 0 }},
		{name: "no model", mutate: func(s *agentapp.RunSpec) { s.Ref = domainllm.ModelRef{} }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			spec := b.spec("FAKE: hello", nil, &recordingStream{})
			tc.mutate(&spec)
			if _, err := b.runner.Run(t.Context(), spec); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("Run = %v, want invalid", err)
			}
		})
	}
}

func TestAStreamFailureKeepsTheAnswerAndSaysTheRecordIsIncomplete(t *testing.T) {
	t.Parallel()

	b := newBareRunner(t, 0)
	stream := &recordingStream{err: errors.New(errors.External, "the bus is gone")}

	result, err := b.runner.Run(t.Context(), b.spec("TOOL:pages_tree{}\nFAKE: done", nil, stream))
	if err != nil {
		t.Fatalf("a turn that answered must not fail because its audit could not be written: %v", err)
	}
	if result.Text != "done" {
		t.Fatalf("the answer was discarded: %q", result.Text)
	}

	warned := false
	for _, outcome := range stream.settled() {
		if outcome.Tool == "agent_audit" && strings.Contains(outcome.Error, "the bus is gone") {
			warned = true
		}
	}
	if !warned {
		t.Fatalf("nothing told the client the record is incomplete: %+v", stream.settled())
	}
}

func TestAToolNameTheRegistryDoesNotKnowBecomesAFailedRow(t *testing.T) {
	t.Parallel()

	b := newBareRunner(t, 0)
	stream := &recordingStream{}

	if _, err := b.runner.Run(t.Context(),
		b.spec("TOOL:pages_invent{}\nFAKE: done", nil, stream)); err != nil {
		t.Fatalf("Run: %v", err)
	}

	settled := stream.settled()
	if len(settled) != 1 {
		t.Fatalf("the outcomes are %+v", settled)
	}
	if settled[0].Tool != "pages_invent" || settled[0].Status != string(domainagent.CallError) {
		t.Fatalf("the outcome is %+v", settled[0])
	}
	if settled[0].Error != "pages_invent is not found" {
		t.Fatalf("the row reads %q, not the message the model was given", settled[0].Error)
	}
}
