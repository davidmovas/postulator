package sqlite_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
	"github.com/davidmovas/postulator/internal/kernel/paging"
)

type runFixture struct {
	store *sqlite.Store
	runs  *sqlite.RunRepo
	items *sqlite.RunItemRepo
	execs *sqlite.StepExecRepo
	blobs *sqlite.ArtifactRepo
	log   *sqlite.RunEventRepo
	run   run.Run
	pages []string
}

func newRunFixture(t *testing.T, targets int) runFixture {
	t.Helper()

	store := sqlitetest.Open(t)
	site := sqlitetest.Site(t, store, "shop")

	pages := make([]string, 0, targets)
	for i := range targets {
		page := sqlitetest.Page(t, store, site.ID, "/page-"+string(rune('a'+i))+"/")
		pages = append(pages, page.ID)
	}

	record := run.Run{
		ID: id.New(), SiteID: site.ID, Kind: run.KindGenerate, Status: run.StatusPending, Targets: pages,
		Recipe:     []template.StepSpec{{Name: "generate_body", Enabled: true, Params: map[string]any{"tone": "plain"}}},
		TemplateID: id.New(), TemplateVersion: 3, PublishMode: run.PublishDraft,
		Budget:    run.Budget{MaxUSD: 2.5, MaxTokens: 1000},
		CreatedBy: kctx.ActorUser, DeadlineAt: sqlitetest.Stamp.Add(time.Hour), CreatedAt: sqlitetest.Stamp,
	}

	fixture := runFixture{
		store: store,
		runs:  sqlite.NewRunRepo(store),
		items: sqlite.NewRunItemRepo(store),
		execs: sqlite.NewStepExecRepo(store),
		blobs: sqlite.NewArtifactRepo(store),
		log:   sqlite.NewRunEventRepo(store),
		run:   record,
		pages: pages,
	}
	if err := fixture.runs.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the run: %v", err)
	}
	return fixture
}

func (f runFixture) insertItem(t *testing.T, pageID, step string) run.Item {
	t.Helper()

	item := run.Item{
		ID: id.New(), RunID: f.run.ID, SiteID: f.run.SiteID, TargetID: pageID, Status: run.StatusPending, CurrentStep: step,
		Checkpoint: run.NewCheckpoint(), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := f.items.Insert(t.Context(), item); err != nil {
		t.Fatalf("insert the run item: %v", err)
	}
	return item
}

func TestRunRepoRoundTrip(t *testing.T) {
	t.Parallel()

	fixture := newRunFixture(t, 1)
	stored, err := fixture.runs.Get(t.Context(), fixture.run.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}

	if stored.Kind != run.KindGenerate || stored.TemplateVersion != 3 || stored.Budget.MaxUSD != 2.5 {
		t.Fatalf("Get = %+v", stored)
	}
	if len(stored.Targets) != 1 || stored.Targets[0] != fixture.pages[0] {
		t.Fatalf("Targets = %v", stored.Targets)
	}
	if len(stored.Recipe) != 1 || stored.Recipe[0].Params["tone"] != "plain" {
		t.Fatalf("Recipe = %+v", stored.Recipe)
	}
	if !stored.DeadlineAt.Equal(fixture.run.DeadlineAt) || stored.StartedAt != nil {
		t.Fatalf("times = %v, %v", stored.DeadlineAt, stored.StartedAt)
	}

	started := sqlitetest.Stamp.Add(time.Minute)
	stored.Status = run.StatusRunning
	stored.StartedAt = &started
	stored.Stats = run.Stats{Items: 1, Done: 1, Tokens: 42, USD: 0.5}
	if err = fixture.runs.Update(t.Context(), stored); err != nil {
		t.Fatalf("Update: %v", err)
	}

	updated, err := fixture.runs.Get(t.Context(), stored.ID)
	if err != nil {
		t.Fatalf("Get after update: %v", err)
	}
	if updated.Status != run.StatusRunning || updated.Stats.Tokens != 42 || updated.StartedAt == nil {
		t.Fatalf("Get after update = %+v", updated)
	}

	if _, err = fixture.runs.Get(t.Context(), id.New()); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get of an absent run = %v", err)
	}
	missing := stored
	missing.ID = id.New()
	if err = fixture.runs.Update(t.Context(), missing); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Update of an absent run = %v", err)
	}
	if err = fixture.runs.Insert(t.Context(), fixture.run); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Insert of a duplicate = %v", err)
	}
}

func TestRunRepoListsAndSweeps(t *testing.T) {
	t.Parallel()

	fixture := newRunFixture(t, 1)
	second := fixture.run
	second.ID = id.New()
	second.Status = run.StatusCompleted
	second.Kind = run.KindRelink
	second.CreatedAt = sqlitetest.Stamp.Add(time.Minute)
	second.DeadlineAt = sqlitetest.Stamp.Add(2 * time.Hour)
	if err := fixture.runs.Insert(t.Context(), second); err != nil {
		t.Fatalf("insert the second run: %v", err)
	}

	list, err := fixture.runs.List(t.Context(), run.Query{SiteID: fixture.run.SiteID}, paging.Request{Limit: 1})
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Items) != 1 || list.Items[0].ID != fixture.run.ID || !list.HasMore {
		t.Fatalf("List = %+v", list)
	}

	next, err := fixture.runs.List(t.Context(), run.Query{SiteID: fixture.run.SiteID},
		paging.Request{After: list.Next, Limit: 10})
	if err != nil {
		t.Fatalf("List page two: %v", err)
	}
	if len(next.Items) != 1 || next.Items[0].ID != second.ID {
		t.Fatalf("List page two = %+v", next)
	}

	completed := run.StatusCompleted
	filtered, err := fixture.runs.List(t.Context(), run.Query{Status: &completed, Sort: run.SortStatus, Desc: true},
		paging.Request{Limit: 10})
	if err != nil {
		t.Fatalf("List by status: %v", err)
	}
	if len(filtered.Items) != 1 || filtered.Items[0].ID != second.ID {
		t.Fatalf("List by status = %+v", filtered)
	}

	relink := run.KindRelink
	byKind, err := fixture.runs.List(t.Context(), run.Query{Kind: &relink}, paging.Request{Limit: 10})
	if err != nil || len(byKind.Items) != 1 {
		t.Fatalf("List by kind = %+v, %v", byKind, err)
	}

	active, err := fixture.runs.Active(t.Context())
	if err != nil || len(active) != 1 || active[0].ID != fixture.run.ID {
		t.Fatalf("Active = %+v, %v", active, err)
	}

	stale, err := fixture.runs.PastDeadline(t.Context(), sqlitetest.Stamp.Add(90*time.Minute), 10)
	if err != nil || len(stale) != 1 || stale[0].ID != fixture.run.ID {
		t.Fatalf("PastDeadline = %+v, %v", stale, err)
	}
}

func TestRunItemClaimIsCompareAndSwap(t *testing.T) {
	t.Parallel()

	fixture := newRunFixture(t, 1)
	item := fixture.insertItem(t, fixture.pages[0], "generate_body")
	lease := sqlitetest.Stamp.Add(time.Minute)

	claimed, err := fixture.items.Claim(t.Context(), item.ID, 0, lease, sqlitetest.Stamp)
	if err != nil {
		t.Fatalf("Claim: %v", err)
	}
	if claimed.AdvanceSeq != 1 || claimed.Status != run.StatusRunning || claimed.LeaseUntil == nil {
		t.Fatalf("Claim = %+v", claimed)
	}

	if _, err = fixture.items.Claim(t.Context(), item.ID, 0, lease, sqlitetest.Stamp); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("a stale claim = %v, want a conflict", err)
	}

	claimed.Status = run.StatusCompleted
	claimed.LeaseUntil = nil
	claimed.UpdatedAt = sqlitetest.Stamp.Add(time.Minute)
	if err = run.Set(claimed.Checkpoint, "step", "done"); err != nil {
		t.Fatalf("Set: %v", err)
	}

	persisted, err := fixture.items.Persist(t.Context(), claimed, 0)
	if err != nil || persisted {
		t.Fatalf("Persist against a stale sequence = %v, %v", persisted, err)
	}

	persisted, err = fixture.items.Persist(t.Context(), claimed, 1)
	if err != nil || !persisted {
		t.Fatalf("Persist = %v, %v", persisted, err)
	}

	stored, err := fixture.items.Get(t.Context(), item.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	value, found, err := run.Get[string](stored.Checkpoint, "step")
	if err != nil || !found || value != "done" {
		t.Fatalf("checkpoint = %q, %v, %v", value, found, err)
	}
	if stored.Status != run.StatusCompleted || stored.LeaseUntil != nil {
		t.Fatalf("Get = %+v", stored)
	}

	if _, err = fixture.items.Get(t.Context(), id.New()); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get of an absent item = %v", err)
	}
	if err = fixture.items.Insert(t.Context(), item); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Insert of a duplicate item = %v", err)
	}
}

func TestRunItemSweepQueries(t *testing.T) {
	t.Parallel()

	fixture := newRunFixture(t, 3)
	running := fixture.run
	running.Status = run.StatusRunning
	if err := fixture.runs.Update(t.Context(), running); err != nil {
		t.Fatalf("update the run: %v", err)
	}

	pending := fixture.insertItem(t, fixture.pages[0], "generate_body")
	waiting := fixture.insertItem(t, fixture.pages[1], "generate_body")
	stalled := fixture.insertItem(t, fixture.pages[2], "generate_body")

	wake := sqlitetest.Stamp.Add(time.Minute)
	waiting.Status = run.StatusWaiting
	waiting.WakeAt = &wake
	waiting.UpdatedAt = sqlitetest.Stamp
	if ok, err := fixture.items.Persist(t.Context(), waiting, 0); err != nil || !ok {
		t.Fatalf("persist the waiting item: %v, %v", ok, err)
	}

	lease := sqlitetest.Stamp.Add(-time.Minute)
	stalled.Status = run.StatusRunning
	stalled.LeaseUntil = &lease
	stalled.UpdatedAt = sqlitetest.Stamp
	if ok, err := fixture.items.Persist(t.Context(), stalled, 0); err != nil || !ok {
		t.Fatalf("persist the stalled item: %v, %v", ok, err)
	}

	due, err := fixture.items.Due(t.Context(), sqlitetest.Stamp.Add(2*time.Minute), 10)
	if err != nil || len(due) != 1 || due[0].ID != waiting.ID {
		t.Fatalf("Due = %+v, %v", due, err)
	}
	if early, dueErr := fixture.items.Due(t.Context(), sqlitetest.Stamp, 10); dueErr != nil || len(early) != 0 {
		t.Fatalf("Due before the wake time = %+v, %v", early, dueErr)
	}

	expired, err := fixture.items.Stalled(t.Context(), sqlitetest.Stamp, 10)
	if err != nil || len(expired) != 1 || expired[0].ID != stalled.ID {
		t.Fatalf("Stalled = %+v, %v", expired, err)
	}

	runnable, err := fixture.items.Runnable(t.Context(), sqlitetest.Stamp, 10)
	if err != nil {
		t.Fatalf("Runnable: %v", err)
	}
	if len(runnable) != 1 || runnable[0].ID != pending.ID {
		t.Fatalf("Runnable = %+v", runnable)
	}

	counts, err := fixture.items.Counts(t.Context(), fixture.run.ID)
	if err != nil {
		t.Fatalf("Counts: %v", err)
	}
	if counts[run.StatusPending] != 1 || counts[run.StatusWaiting] != 1 || counts[run.StatusRunning] != 1 {
		t.Fatalf("Counts = %v", counts)
	}

	for _, tc := range []struct {
		item run.Item
		from run.Status
	}{{item: waiting, from: run.StatusWaiting}, {item: stalled, from: run.StatusRunning}} {
		requeued, requeueErr := fixture.items.Requeue(t.Context(), tc.item.ID, 0, tc.from, sqlitetest.Stamp)
		if requeueErr != nil || !requeued {
			t.Fatalf("Requeue(%s) = %v, %v", tc.from, requeued, requeueErr)
		}
		again, staleErr := fixture.items.Requeue(t.Context(), tc.item.ID, 0, tc.from, sqlitetest.Stamp)
		if staleErr != nil || again {
			t.Fatalf("Requeue against a stale sequence = %v, %v", again, staleErr)
		}
	}
	if woken, wokenErr := fixture.items.Runnable(t.Context(), sqlitetest.Stamp, 10); wokenErr != nil || len(woken) != 3 {
		t.Fatalf("Runnable after the sweep = %+v, %v", woken, wokenErr)
	}

	active := []run.Status{run.StatusPending, run.StatusRunning, run.StatusWaiting, run.StatusPaused}
	stopped, err := fixture.items.StopAll(t.Context(), fixture.run.ID, active, run.StatusPaused, run.PauseUser, sqlitetest.Stamp)
	if err != nil || stopped != 3 {
		t.Fatalf("StopAll = %d, %v", stopped, err)
	}
	resumed, err := fixture.items.ResumeAll(t.Context(), fixture.run.ID, sqlitetest.Stamp)
	if err != nil || resumed != 3 {
		t.Fatalf("ResumeAll = %d, %v", resumed, err)
	}

	cancelled, err := fixture.items.StopAll(t.Context(), fixture.run.ID, active, run.StatusCancelled, "", sqlitetest.Stamp)
	if err != nil || cancelled != 3 {
		t.Fatalf("StopAll to a terminal status = %d, %v", cancelled, err)
	}
	after, err := fixture.items.Get(t.Context(), pending.ID)
	if err != nil || after.FinishedAt == nil || after.Status != run.StatusCancelled {
		t.Fatalf("item after cancellation = %+v, %v", after, err)
	}

	list, err := fixture.items.List(t.Context(), run.ItemQuery{RunID: fixture.run.ID}, paging.Request{Limit: 2})
	if err != nil || len(list.Items) != 2 || !list.HasMore {
		t.Fatalf("List = %+v, %v", list, err)
	}
	terminal := run.StatusCancelled
	filtered, err := fixture.items.List(t.Context(), run.ItemQuery{RunID: fixture.run.ID, Status: &terminal, Desc: true},
		paging.Request{Limit: 10})
	if err != nil || len(filtered.Items) != 3 {
		t.Fatalf("List by status = %+v, %v", filtered, err)
	}

	all, err := fixture.items.ByRun(t.Context(), fixture.run.ID)
	if err != nil || len(all) != 3 {
		t.Fatalf("ByRun = %+v, %v", all, err)
	}
}

func TestArtifactRepoReplacesAndPurges(t *testing.T) {
	t.Parallel()

	fixture := newRunFixture(t, 1)
	item := fixture.insertItem(t, fixture.pages[0], "generate_body")

	first, err := run.NewArtifact(run.Artifact{
		ID: id.New(), RunID: fixture.run.ID, ItemID: item.ID, Step: "generate_body",
		Kind: run.ArtifactBodyHTML, Blob: []byte("<p>one</p>"), CreatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	if err = fixture.blobs.ReplaceStep(t.Context(), item.ID, "generate_body", []run.Artifact{first}); err != nil {
		t.Fatalf("ReplaceStep: %v", err)
	}

	stored, err := fixture.blobs.Get(t.Context(), first.ID)
	if err != nil || string(stored.Blob) != "<p>one</p>" || stored.Hash != first.Hash {
		t.Fatalf("Get = %+v, %v", stored, err)
	}

	second := first
	second.ID = id.New()
	second.Blob = []byte("<p>two</p>")
	second.Size = len(second.Blob)
	second.Hash = run.HashBlob(second.Blob)
	if err = fixture.blobs.ReplaceStep(t.Context(), item.ID, "generate_body", []run.Artifact{second}); err != nil {
		t.Fatalf("ReplaceStep of the retry: %v", err)
	}

	remaining, err := fixture.blobs.ByItem(t.Context(), item.ID)
	if err != nil || len(remaining) != 1 || remaining[0].ID != second.ID {
		t.Fatalf("ByItem = %+v, %v", remaining, err)
	}
	if _, err = fixture.blobs.Get(t.Context(), first.ID); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get of a replaced artifact = %v", err)
	}

	published, err := run.NewArtifact(run.Artifact{
		ID: id.New(), RunID: fixture.run.ID, ItemID: item.ID, Step: "publish",
		Kind: run.ArtifactPublishResult, Blob: []byte(`{"wpId":7}`), CreatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	if err = fixture.blobs.ReplaceStep(t.Context(), item.ID, "publish", []run.Artifact{published}); err != nil {
		t.Fatalf("ReplaceStep of the publish result: %v", err)
	}

	purged, err := fixture.blobs.PurgePublishedBefore(t.Context(), sqlitetest.Stamp.Add(time.Hour))
	if err != nil || purged != 1 {
		t.Fatalf("PurgePublishedBefore = %d, %v", purged, err)
	}

	body, err := fixture.blobs.Get(t.Context(), second.ID)
	if err != nil || !body.Purged || len(body.Blob) != 0 || body.Size != 0 {
		t.Fatalf("purged artifact = %+v, %v", body, err)
	}
	report, err := fixture.blobs.Get(t.Context(), published.ID)
	if err != nil || report.Purged {
		t.Fatalf("the publish result must survive the purge: %+v, %v", report, err)
	}
}

func TestStepExecRepo(t *testing.T) {
	t.Parallel()

	fixture := newRunFixture(t, 1)
	item := fixture.insertItem(t, fixture.pages[0], "generate_body")
	finished := sqlitetest.Stamp.Add(time.Second)

	exec := run.StepExec{
		ID: id.New(), RunID: fixture.run.ID, ItemID: item.ID, Step: "generate_body", Attempt: 1,
		Status: run.ExecDone, InputHash: "hash-1", Tokens: 100, USD: 0.02,
		StartedAt: sqlitetest.Stamp, FinishedAt: &finished,
	}
	if err := fixture.execs.Insert(t.Context(), exec); err != nil {
		t.Fatalf("Insert: %v", err)
	}
	if err := fixture.execs.Insert(t.Context(), exec); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("Insert of a duplicate attempt = %v", err)
	}

	bad := exec
	bad.ID = id.New()
	bad.Status = "sleeping"
	if err := fixture.execs.Insert(t.Context(), bad); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Insert with an unknown status = %v", err)
	}

	done, err := fixture.execs.Done(t.Context(), item.ID, "generate_body", "hash-1")
	if err != nil || done.Tokens != 100 || done.FinishedAt == nil {
		t.Fatalf("Done = %+v, %v", done, err)
	}
	if _, err = fixture.execs.Done(t.Context(), item.ID, "generate_body", "hash-2"); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Done for another input = %v", err)
	}

	all, err := fixture.execs.ByItem(t.Context(), item.ID)
	if err != nil || len(all) != 1 {
		t.Fatalf("ByItem = %+v, %v", all, err)
	}
}

func TestRunEventLogIsGapless(t *testing.T) {
	t.Parallel()

	fixture := newRunFixture(t, 1)
	types := []string{"run.queued", "run.started", "item.started"}
	for i, eventType := range types {
		event, err := fixture.log.Append(t.Context(), fixture.run.ID, eventType, sqlitetest.Stamp,
			json.RawMessage(`{"runId":"`+fixture.run.ID+`"}`))
		if err != nil {
			t.Fatalf("Append: %v", err)
		}
		if event.Seq != int64(i+1) {
			t.Fatalf("Seq = %d, want %d", event.Seq, i+1)
		}
	}

	listed, err := fixture.log.List(t.Context(), fixture.run.ID, 0, 10)
	if err != nil || len(listed) != 3 {
		t.Fatalf("List = %+v, %v", listed, err)
	}
	for i, event := range listed {
		if event.Seq != int64(i+1) || event.Type != types[i] {
			t.Fatalf("event %d = %+v", i, event)
		}
		if !event.At.Equal(sqlitetest.Stamp) {
			t.Fatalf("event %d timestamp = %v", i, event.At)
		}
	}

	after, err := fixture.log.List(t.Context(), fixture.run.ID, 2, 10)
	if err != nil || len(after) != 1 || after[0].Seq != 3 {
		t.Fatalf("List since 2 = %+v, %v", after, err)
	}

	if _, err = fixture.log.Append(t.Context(), fixture.run.ID, "", sqlitetest.Stamp, nil); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Append without a type = %v", err)
	}
}

func TestRunEventRepoPurgesTheEventsOfFinishedRuns(t *testing.T) {
	t.Parallel()

	fixture := newRunFixture(t, 1)
	other := fixture.run
	other.ID = id.New()
	other.Status = run.StatusRunning
	if err := fixture.runs.Insert(t.Context(), other); err != nil {
		t.Fatalf("insert the second run: %v", err)
	}

	for _, runID := range []string{fixture.run.ID, other.ID} {
		for _, eventType := range []string{"run.started", "run.progress"} {
			if _, err := fixture.log.Append(t.Context(), runID, eventType, sqlitetest.Stamp, []byte(`{}`)); err != nil {
				t.Fatalf("append an event: %v", err)
			}
		}
	}

	at := sqlitetest.Stamp
	finished := fixture.run
	finished.Status = run.StatusCompleted
	finished.FinishedAt = &at
	if err := fixture.runs.Update(t.Context(), finished); err != nil {
		t.Fatalf("finish the run: %v", err)
	}

	dropped, err := fixture.log.PurgeTerminalBefore(t.Context(), sqlitetest.Stamp.Add(-time.Hour))
	if err != nil || dropped != 0 {
		t.Fatalf("PurgeTerminalBefore before the cutoff = %d, %v", dropped, err)
	}

	dropped, err = fixture.log.PurgeTerminalBefore(t.Context(), sqlitetest.Stamp.Add(time.Hour))
	if err != nil || dropped != 2 {
		t.Fatalf("PurgeTerminalBefore = %d, %v, want 2", dropped, err)
	}

	gone, err := fixture.log.List(t.Context(), fixture.run.ID, 0, 10)
	if err != nil || len(gone) != 0 {
		t.Fatalf("the finished run still holds %d events, %v", len(gone), err)
	}

	kept, err := fixture.log.List(t.Context(), other.ID, 0, 10)
	if err != nil || len(kept) != 2 {
		t.Fatalf("the running run holds %d events, %v, want 2", len(kept), err)
	}
}
