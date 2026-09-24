package runtime_test

import (
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/domain/run"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/runtime"
)

func TestTheSweepPurgesArtifactsAndEventsPastTheirRetention(t *testing.T) {
	t.Parallel()

	h := newHarness(t, 1)
	old := time.Now().UTC().Add(-10 * 24 * time.Hour)

	finished := h.newRun(recipeOf("generate_body"))
	finished.Status = run.StatusCompleted
	finished.CreatedAt = old
	finished.DeadlineAt = old.Add(time.Hour)
	finished.FinishedAt = &old
	finished.CreatedBy = kctx.ActorUser
	if err := h.runs.Insert(t.Context(), finished); err != nil {
		t.Fatalf("insert the finished run: %v", err)
	}

	item := run.Item{
		ID: id.New(), RunID: finished.ID, SiteID: finished.SiteID, TargetID: h.pages[0],
		Status: run.StatusCompleted, CurrentStep: "publish", Checkpoint: run.NewCheckpoint(),
		CreatedAt: old, UpdatedAt: old,
	}
	if err := h.items.Insert(t.Context(), item); err != nil {
		t.Fatalf("insert the run item: %v", err)
	}

	body := artifact(t, finished.ID, item.ID, "generate_body", run.ArtifactBodyHTML, `<p>the draft body</p>`, old)
	report := artifact(t, finished.ID, item.ID, "report", run.ArtifactFinalReport, `{"compliance":1}`, old)
	published := artifact(t, finished.ID, item.ID, "publish", run.ArtifactPublishResult, `{"wpId":7}`, old)
	for step, blobs := range map[string][]run.Artifact{
		"generate_body": {body},
		"report":        {report},
		"publish":       {published},
	} {
		if err := h.blobs.ReplaceStep(t.Context(), item.ID, step, blobs); err != nil {
			t.Fatalf("store the %s artifact: %v", step, err)
		}
	}

	for _, eventType := range []string{"run.started", "run.completed"} {
		if _, err := h.log.Append(t.Context(), finished.ID, eventType, old, []byte(`{}`)); err != nil {
			t.Fatalf("append an event: %v", err)
		}
	}

	running := h.newRun(recipeOf("generate_body"))
	running.Status = run.StatusRunning
	running.CreatedBy = kctx.ActorUser
	running.DeadlineAt = time.Now().UTC().Add(time.Hour)
	if err := h.runs.Insert(t.Context(), running); err != nil {
		t.Fatalf("insert the running run: %v", err)
	}
	if _, err := h.log.Append(t.Context(), running.ID, "run.started", old, []byte(`{}`)); err != nil {
		t.Fatalf("append the event of the running run: %v", err)
	}

	engine := runtime.New(runtime.Deps{
		Runs: h.runs, Items: h.items, Artifacts: h.blobs, Execs: h.execs, Events: h.log,
		Pages: sqlite.NewPageRepo(h.store), Specs: h.specs, Spend: h.spend,
		Keys: h.keys, Catalog: &stubCatalog{}, Profiles: &stubProfiles{}, UnitOfWork: h.store, Publisher: h.bus,
	}, mustRegister(t, bodyStep(newCounter())), runtime.Config{
		Workers: 1, PerSite: 1, SweepInterval: 20 * time.Millisecond,
		RetentionDays: 1, EventRetentionDays: 1, RunDeadline: time.Hour,
	}, systemClock(), h.logger)

	if err := engine.Start(t.Context()); err != nil {
		t.Fatalf("start the engine: %v", err)
	}
	t.Cleanup(engine.Stop)

	waitFor(t, "the sweep to purge the draft body", func() bool {
		stored, err := h.blobs.Get(t.Context(), body.ID)
		return err == nil && stored.Purged
	})
	waitFor(t, "the sweep to purge the events of the finished run", func() bool {
		events, err := h.log.List(t.Context(), finished.ID, 0, 10)
		return err == nil && len(events) == 0
	})

	kept, err := h.blobs.Get(t.Context(), report.ID)
	if err != nil || kept.Purged || string(kept.Blob) != `{"compliance":1}` {
		t.Fatalf("the final report must survive the purge: %+v, %v", kept, err)
	}

	live, err := h.log.List(t.Context(), running.ID, 0, 10)
	if err != nil || len(live) != 1 {
		t.Fatalf("the events of a run still going hold %d rows, %v, want 1", len(live), err)
	}
}

func artifact(t *testing.T, runID, itemID, step string, kind run.ArtifactKind, blob string, at time.Time) run.Artifact {
	t.Helper()

	made, err := run.NewArtifact(run.Artifact{
		ID: id.New(), RunID: runID, ItemID: itemID, Step: step, Kind: kind, Blob: []byte(blob), CreatedAt: at,
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	return made
}
