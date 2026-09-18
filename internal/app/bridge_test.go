package app_test

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"

	"github.com/davidmovas/postulator/internal/app"
	"github.com/davidmovas/postulator/internal/application/events"
	"github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/application/pages"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/sites"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/transport/wails"
)

const (
	runDeadline = 30 * time.Second
	runPoll     = 20 * time.Millisecond
)

type recordingEmitter struct {
	mu        sync.Mutex
	envelopes []events.Envelope
}

func (r *recordingEmitter) Emit(_ string, data ...any) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, item := range data {
		envelope, ok := item.(events.Envelope)
		if !ok {
			continue
		}
		r.envelopes = append(r.envelopes, envelope)
	}
	return true
}

func (r *recordingEmitter) forRun(runID string) []events.Envelope {
	r.mu.Lock()
	defer r.mu.Unlock()

	collected := make([]events.Envelope, 0, len(r.envelopes))
	for _, envelope := range r.envelopes {
		if envelope.RunID != nil && *envelope.RunID == runID {
			collected = append(collected, envelope)
		}
	}
	return collected
}

func plannedPage(t *testing.T, core *app.Core) (siteID, pageID string) {
	t.Helper()

	owner, err := core.Sites.Create(t.Context(), sites.CreateRequest{
		Name: "bridge", BaseURL: "https://bridge.test", Username: "editor", Password: "hunter2",
	})
	if err != nil {
		t.Fatalf("create the site: %v", err)
	}

	seeded, err := core.Templates.ListTemplates(t.Context(), templates.ListTemplatesRequest{Scope: "global"})
	if err != nil || len(seeded.Items) == 0 {
		t.Fatalf("list the seeded templates: %d, %v", len(seeded.Items), err)
	}

	entity, err := core.Graph.CreateEntity(t.Context(), graph.CreateEntityRequest{
		SiteID: owner.Site.ID, Name: "running shoes", Kind: "topic", PrimaryKeyword: "running shoes",
	})
	if err != nil {
		t.Fatalf("create the entity: %v", err)
	}

	page, err := core.Pages.Create(t.Context(), pages.CreateRequest{
		SiteID:     owner.Site.ID,
		Path:       "/running-shoes/",
		Title:      "Running shoes",
		H1:         "Running shoes",
		EntityID:   &entity.Entity.ID,
		TemplateID: &seeded.Items[0].ID,
	})
	if err != nil {
		t.Fatalf("create the page: %v", err)
	}
	return owner.Site.ID, page.Page.ID
}

func settled(t *testing.T, service *wails.RunsService, runID string) runs.Run {
	t.Helper()

	deadline := time.Now().Add(runDeadline)
	for {
		current, err := service.Get(t.Context(), runs.GetRequest{RunID: runID})
		if err != nil {
			t.Fatalf("read the run: %v", err)
		}
		if !current.Run.FinishedAt.Std().IsZero() {
			return current.Run
		}
		if time.Now().After(deadline) {
			t.Fatalf("the run is still %q after %s", current.Run.Status, runDeadline)
		}
		time.Sleep(runPoll)
	}
}

func TestTheEventBridgeCarriesARunFromTheEngineToTheWindow(t *testing.T) {
	t.Parallel()

	home := t.TempDir()
	core, err := app.Open(t.Context(), app.Config{
		DatabasePath: filepath.Join(home, "postulator.db"), KeyDir: home,
	}, zaptest.NewLogger(t))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := core.Close(); closeErr != nil {
			t.Errorf("Close: %v", closeErr)
		}
	})

	emitter := &recordingEmitter{}
	if connectErr := core.Events.Connect(emitter, clock.System{}); connectErr != nil {
		t.Fatalf("Connect: %v", connectErr)
	}

	siteID, pageID := plannedPage(t, core)
	service := wails.NewRunsService(zaptest.NewLogger(t), core.Runs)

	started, err := service.Start(t.Context(), runs.StartRequest{
		SiteID:  siteID,
		PageIDs: []string{pageID},
		Recipe:  []template.StepSpec{{Name: "resolve_context", Enabled: true}},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if started.RunID == "" {
		t.Fatal("Start returned no run id")
	}

	final := settled(t, service, started.RunID)
	if final.Status != "completed" {
		t.Fatalf("the run finished as %q, want completed", final.Status)
	}

	live := emitter.forRun(started.RunID)
	if len(live) < 2 {
		t.Fatalf("the bridge delivered %d run events, want the whole run", len(live))
	}
	if live[0].Type != events.RunQueued {
		t.Errorf("the first live event is %q, want %q", live[0].Type, events.RunQueued)
	}
	for index, envelope := range live {
		if envelope.Seq != int64(index+1) {
			t.Fatalf("live event %d carries seq %d, want a gapless sequence", index, envelope.Seq)
		}
		if !strings.Contains(string(envelope.Type), ".") {
			t.Errorf("live event %d carries the type %q", index, envelope.Type)
		}
	}

	logged, err := service.ListEvents(t.Context(), runs.ListEventsRequest{RunID: started.RunID})
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	if len(logged.Events) != len(live) {
		t.Fatalf("the durable log holds %d events and the bridge delivered %d", len(logged.Events), len(live))
	}
	for index, recorded := range logged.Events {
		if recorded.Seq != live[index].Seq || recorded.Type != string(live[index].Type) {
			t.Fatalf("event %d is %q/%d in the log and %q/%d live",
				index, recorded.Type, recorded.Seq, live[index].Type, live[index].Seq)
		}
	}

	tail, err := service.ListEvents(t.Context(), runs.ListEventsRequest{RunID: started.RunID, SinceSeq: live[0].Seq})
	if err != nil {
		t.Fatalf("ListEvents after a sequence: %v", err)
	}
	if len(tail.Events) != len(logged.Events)-1 || tail.Events[0].Seq != live[1].Seq {
		t.Fatalf("the catch-up from seq %d returned %d events starting at %d",
			live[0].Seq, len(tail.Events), tail.Events[0].Seq)
	}
}
