package runs_test

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/runs"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/dto"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type fakeEngine struct {
	queued     run.Run
	estimate   run.Estimate
	enqueueErr error
	failWith   error
	paused     string
	reason     run.PauseReason
	resumed    string
	cancelled  string
	retried    string
}

func (f *fakeEngine) Enqueue(_ context.Context, record run.Run) (run.Run, error) {
	if f.enqueueErr != nil {
		return run.Run{}, f.enqueueErr
	}
	f.queued = record
	return record, nil
}

func (f *fakeEngine) EstimateRun(context.Context, run.Run, template.TemplateSpec) (run.Estimate, error) {
	if f.failWith != nil {
		return run.Estimate{}, f.failWith
	}
	return f.estimate, nil
}

func (f *fakeEngine) Pause(_ context.Context, runID string, reason run.PauseReason) error {
	f.paused, f.reason = runID, reason
	return f.failWith
}

func (f *fakeEngine) Resume(_ context.Context, runID string) error {
	f.resumed = runID
	return f.failWith
}

func (f *fakeEngine) Cancel(_ context.Context, runID string) error {
	f.cancelled = runID
	return f.failWith
}

func (f *fakeEngine) RetryStep(_ context.Context, itemID string) error {
	f.retried = itemID
	return f.failWith
}

type fakeSpecs struct {
	spec    template.TemplateSpec
	siteID  string
	version int
	seen    []string
	err     error
}

func (f *fakeSpecs) ResolveForPage(_ context.Context, req templates.ResolveForPageRequest) (templates.ResolveForPageResponse, error) {
	if f.err != nil {
		return templates.ResolveForPageResponse{}, f.err
	}
	f.seen = append(f.seen, req.PageID)
	return templates.ResolveForPageResponse{
		TemplateID: "template-1", SiteID: f.siteID, Version: f.version, Spec: f.spec,
	}, nil
}

type fixture struct {
	service *runs.Service
	engine  *fakeEngine
	specs   *fakeSpecs
	runs    *sqlite.RunRepo
	items   *sqlite.RunItemRepo
	blobs   *sqlite.ArtifactRepo
	log     *sqlite.RunEventRepo
	siteID  string
	pages   []string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	store := sqlitetest.Open(t)
	site := sqlitetest.Site(t, store, "shop")

	pages := make([]string, 0, 2)
	for _, path := range []string{"/hub/", "/hub/child/"} {
		pages = append(pages, sqlitetest.Page(t, store, site.ID, path).ID)
	}

	engine := &fakeEngine{estimate: run.Estimate{Tokens: 4200, USD: 0.12}}
	specs := &fakeSpecs{
		siteID:  "s",
		version: 2,
		spec: template.TemplateSpec{
			Recipe: []template.StepSpec{{Name: "generate_body", Enabled: true}},
		},
	}

	runRepo := sqlite.NewRunRepo(store)
	itemRepo := sqlite.NewRunItemRepo(store)
	blobRepo := sqlite.NewArtifactRepo(store)
	logRepo := sqlite.NewRunEventRepo(store)

	return &fixture{
		service: runs.New(engine, runRepo, itemRepo, blobRepo, logRepo, specs),
		engine:  engine,
		specs:   specs,
		runs:    runRepo,
		items:   itemRepo,
		blobs:   blobRepo,
		log:     logRepo,
		siteID:  site.ID,
		pages:   pages,
	}
}

func (f *fixture) seedRun(t *testing.T, status run.Status) (run.Run, run.Item) {
	t.Helper()

	record := run.Run{
		ID: id.New(), SiteID: f.siteID, Kind: run.KindGenerate, Status: status, Targets: f.pages,
		Recipe:     []template.StepSpec{{Name: "generate_body", Enabled: true}},
		TemplateID: "template-1", TemplateVersion: 2, PublishMode: run.PublishDraft,
		CreatedBy: kctx.ActorUser, DeadlineAt: sqlitetest.Stamp.Add(time.Hour), CreatedAt: sqlitetest.Stamp,
	}
	if err := f.runs.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the run: %v", err)
	}

	item := run.Item{
		ID: id.New(), RunID: record.ID, SiteID: record.SiteID, TargetID: f.pages[0], Status: run.StatusCompleted,
		CurrentStep: "generate_body", Checkpoint: run.NewCheckpoint(),
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := f.items.Insert(t.Context(), item); err != nil {
		t.Fatalf("insert the run item: %v", err)
	}
	return record, item
}

func TestStartResolvesEveryTargetAndEstimates(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	fixture.specs.siteID = fixture.siteID
	ctx := kctx.WithActor(t.Context(), kctx.ActorAgent)

	resp, err := fixture.service.Start(ctx, runs.StartRequest{
		SiteID:  fixture.siteID,
		PageIDs: []string{fixture.pages[0], fixture.pages[1], fixture.pages[0], "  "},
		Budget:  run.Budget{MaxUSD: 1},
	})
	if err != nil {
		t.Fatalf("Start: %v", err)
	}
	if resp.RunID == "" || resp.Estimate.USD != 0.12 {
		t.Fatalf("Start = %+v", resp)
	}
	if len(fixture.specs.seen) != 2 {
		t.Fatalf("the template was resolved for %v", fixture.specs.seen)
	}

	queued := fixture.engine.queued
	if len(queued.Targets) != 2 {
		t.Fatalf("Targets = %v, want the two distinct pages", queued.Targets)
	}
	if queued.TemplateID != "template-1" || queued.TemplateVersion != 2 {
		t.Fatalf("template = %s@%d", queued.TemplateID, queued.TemplateVersion)
	}
	if queued.CreatedBy != kctx.ActorAgent {
		t.Fatalf("CreatedBy = %q", queued.CreatedBy)
	}
	if len(queued.Recipe) != 1 || queued.Recipe[0].Name != "generate_body" {
		t.Fatalf("Recipe = %+v", queued.Recipe)
	}
	if queued.PublishMode != run.PublishDraft || queued.Kind != run.KindGenerate {
		t.Fatalf("defaults = %q, %q", queued.PublishMode, queued.Kind)
	}
}

func TestStartRejectsBadRequests(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		request runs.StartRequest
		prepare func(*fixture)
		want    errors.Code
	}{
		{name: "no site", request: runs.StartRequest{PageIDs: []string{"p"}}, want: errors.Invalid},
		{name: "no pages", request: runs.StartRequest{SiteID: "s"}, want: errors.Invalid},
		{
			name:    "blank pages",
			request: runs.StartRequest{SiteID: "s", PageIDs: []string{" ", ""}},
			want:    errors.Invalid,
		},
		{
			name:    "unknown kind",
			request: runs.StartRequest{SiteID: "s", PageIDs: []string{"p"}, Kind: "dance"},
			want:    errors.Invalid,
		},
		{
			name:    "unknown publish mode",
			request: runs.StartRequest{SiteID: "s", PageIDs: []string{"p"}, PublishMode: "broadcast"},
			want:    errors.Invalid,
		},
		{
			name:    "the page has no template",
			request: runs.StartRequest{SiteID: "s", PageIDs: []string{"p"}},
			prepare: func(f *fixture) { f.specs.err = errors.New(errors.NotFound, "no template") },
			want:    errors.NotFound,
		},
		{
			name:    "no recipe anywhere",
			request: runs.StartRequest{SiteID: "s", PageIDs: []string{"p"}},
			prepare: func(f *fixture) { f.specs.spec = template.TemplateSpec{} },
			want:    errors.Invalid,
		},
		{
			name:    "the estimate fails",
			request: runs.StartRequest{SiteID: "s", PageIDs: []string{"p"}},
			prepare: func(f *fixture) { f.engine.failWith = errors.New(errors.NotFound, "no model") },
			want:    errors.NotFound,
		},
		{
			name:    "the engine refuses the recipe",
			request: runs.StartRequest{SiteID: "s", PageIDs: []string{"p"}},
			prepare: func(f *fixture) { f.engine.enqueueErr = errors.New(errors.Invalid, "broken recipe") },
			want:    errors.Invalid,
		},
		{
			name:    "a target page belongs to another site",
			request: runs.StartRequest{SiteID: "s", PageIDs: []string{"p"}},
			prepare: func(f *fixture) { f.specs.siteID = "other" },
			want:    errors.Invalid,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			fixture := newFixture(t)
			if tc.prepare != nil {
				tc.prepare(fixture)
			}
			if _, err := fixture.service.Start(t.Context(), tc.request); !errors.IsCode(err, tc.want) {
				t.Fatalf("Start = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestGetAndListReadTheSnapshots(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	record, item := fixture.seedRun(t, run.StatusRunning)

	got, err := fixture.service.Get(t.Context(), runs.GetRequest{RunID: record.ID})
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Run.ID != record.ID || got.Run.Status != "running" || got.Run.StartedAt.String() != "" {
		t.Fatalf("Get = %+v", got.Run)
	}
	if got.Run.DeadlineAt.String() == "" || len(got.Run.Targets) != 2 {
		t.Fatalf("Get = %+v", got.Run)
	}
	if _, err = fixture.service.Get(t.Context(), runs.GetRequest{RunID: id.New()}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("Get of an absent run = %v", err)
	}

	list, err := fixture.service.List(t.Context(), runs.ListRequest{SiteID: fixture.siteID, Status: "running"})
	if err != nil || len(list.Items) != 1 || list.Items[0].ID != record.ID {
		t.Fatalf("List = %+v, %v", list, err)
	}

	byKind, err := fixture.service.List(t.Context(), runs.ListRequest{
		Kind:        "generate",
		ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "status", Desc: true}},
	})
	if err != nil || len(byKind.Items) != 1 {
		t.Fatalf("List by kind = %+v, %v", byKind, err)
	}

	items, err := fixture.service.ListItems(t.Context(), runs.ListItemsRequest{RunID: record.ID, Status: "completed"})
	if err != nil || len(items.Items) != 1 || items.Items[0].ID != item.ID {
		t.Fatalf("ListItems = %+v, %v", items, err)
	}
	if items.Items[0].CurrentStep != "generate_body" || items.Items[0].TargetID != fixture.pages[0] {
		t.Fatalf("ListItems = %+v", items.Items[0])
	}
}

func TestListRejectsUnknownFilters(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	cases := []struct {
		name    string
		request runs.ListRequest
	}{
		{name: "unknown status", request: runs.ListRequest{Status: "sleeping"}},
		{name: "unknown kind", request: runs.ListRequest{Kind: "dance"}},
		{
			name:    "unknown sort",
			request: runs.ListRequest{ListRequest: dto.ListRequest{Sort: &dto.Sort{Field: "brightness"}}},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := fixture.service.List(t.Context(), tc.request); !errors.IsCode(err, errors.Invalid) {
				t.Fatalf("List = %v, want an invalid error", err)
			}
		})
	}

	if _, err := fixture.service.ListItems(t.Context(), runs.ListItemsRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("ListItems without a run = %v", err)
	}
	if _, err := fixture.service.ListItems(t.Context(), runs.ListItemsRequest{RunID: "r", Status: "sleeping"}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("ListItems with an unknown status = %v", err)
	}
}

func TestListEventsPagesBySequence(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	record, _ := fixture.seedRun(t, run.StatusRunning)

	for _, eventType := range []string{"run.queued", "run.started", "item.started"} {
		if _, err := fixture.log.Append(t.Context(), record.ID, eventType, sqlitetest.Stamp,
			json.RawMessage(`{"runId":"`+record.ID+`"}`)); err != nil {
			t.Fatalf("append %s: %v", eventType, err)
		}
	}

	all, err := fixture.service.ListEvents(t.Context(), runs.ListEventsRequest{RunID: record.ID})
	if err != nil || len(all.Events) != 3 {
		t.Fatalf("ListEvents = %+v, %v", all, err)
	}
	if all.Events[0].Seq != 1 || all.Events[0].Type != "run.queued" || all.Events[0].At.String() == "" {
		t.Fatalf("the first event = %+v", all.Events[0])
	}

	after, err := fixture.service.ListEvents(t.Context(), runs.ListEventsRequest{RunID: record.ID, SinceSeq: 2, Limit: 10})
	if err != nil || len(after.Events) != 1 || after.Events[0].Seq != 3 {
		t.Fatalf("ListEvents since 2 = %+v, %v", after, err)
	}

	capped, err := fixture.service.ListEvents(t.Context(), runs.ListEventsRequest{RunID: record.ID, Limit: 10_000})
	if err != nil || len(capped.Events) != 3 {
		t.Fatalf("ListEvents with an oversized limit = %+v, %v", capped, err)
	}

	if _, err = fixture.service.ListEvents(t.Context(), runs.ListEventsRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("ListEvents without a run = %v", err)
	}
	if _, err = fixture.service.ListEvents(t.Context(), runs.ListEventsRequest{RunID: "r", SinceSeq: -1}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("ListEvents with a negative sequence = %v", err)
	}
}

func TestGetArtifactReadsTheLatestOfItsKind(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	record, item := fixture.seedRun(t, run.StatusCompleted)

	first, err := run.NewArtifact(run.Artifact{
		ID: id.New(), RunID: record.ID, ItemID: item.ID, Step: "generate_body",
		Kind: run.ArtifactBodyHTML, Blob: []byte("<p>draft</p>"), CreatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	second, err := run.NewArtifact(run.Artifact{
		ID: id.New(), RunID: record.ID, ItemID: item.ID, Step: "insert_links",
		Kind: run.ArtifactBodyHTML, Blob: []byte("<p>linked</p>"), CreatedAt: sqlitetest.Stamp.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	for _, artifact := range []run.Artifact{first, second} {
		if err = fixture.blobs.ReplaceStep(t.Context(), item.ID, artifact.Step, []run.Artifact{artifact}); err != nil {
			t.Fatalf("ReplaceStep: %v", err)
		}
	}

	got, err := fixture.service.GetArtifact(t.Context(), runs.GetArtifactRequest{ItemID: item.ID, Kind: "body_html"})
	if err != nil {
		t.Fatalf("GetArtifact: %v", err)
	}
	if got.Artifact.Content != "<p>linked</p>" || got.Artifact.Step != "insert_links" {
		t.Fatalf("GetArtifact = %+v", got.Artifact)
	}
	if got.Artifact.Hash != second.Hash || got.Artifact.Size != second.Size {
		t.Fatalf("GetArtifact = %+v", got.Artifact)
	}

	cases := []struct {
		name    string
		request runs.GetArtifactRequest
		want    errors.Code
	}{
		{name: "no item", request: runs.GetArtifactRequest{Kind: "body_html"}, want: errors.Invalid},
		{name: "unknown kind", request: runs.GetArtifactRequest{ItemID: item.ID, Kind: "poem"}, want: errors.Invalid},
		{name: "absent kind", request: runs.GetArtifactRequest{ItemID: item.ID, Kind: "meta"}, want: errors.NotFound},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := fixture.service.GetArtifact(t.Context(), tc.request); !errors.IsCode(err, tc.want) {
				t.Fatalf("GetArtifact = %v, want %s", err, tc.want)
			}
		})
	}
}

func TestListArtifactsReportsWhatExistsWithoutItsBlob(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	record, item := fixture.seedRun(t, run.StatusCompleted)
	expires := sqlitetest.Stamp.Add(720 * time.Hour)

	body, err := run.NewArtifact(run.Artifact{
		ID: id.New(), RunID: record.ID, ItemID: item.ID, Step: "generate_body",
		Kind: run.ArtifactBodyHTML, Blob: []byte("<p>draft</p>"), CreatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	body.ExpiresAt = &expires

	report, err := run.NewArtifact(run.Artifact{
		ID: id.New(), RunID: record.ID, ItemID: item.ID, Step: "report",
		Kind: run.ArtifactFinalReport, Blob: []byte(`{"ok":true}`), CreatedAt: sqlitetest.Stamp.Add(time.Minute),
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}

	purged, err := run.NewArtifact(run.Artifact{
		ID: id.New(), RunID: record.ID, ItemID: item.ID, Step: "generate_body",
		Kind: run.ArtifactDraft, Blob: []byte("gone"), CreatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewArtifact: %v", err)
	}
	purged.Blob, purged.Size, purged.Purged, purged.ExpiresAt = nil, 0, true, &expires

	if err = fixture.blobs.ReplaceStep(t.Context(), item.ID, "generate_body", []run.Artifact{body, purged}); err != nil {
		t.Fatalf("ReplaceStep: %v", err)
	}
	if err = fixture.blobs.ReplaceStep(t.Context(), item.ID, "report", []run.Artifact{report}); err != nil {
		t.Fatalf("ReplaceStep: %v", err)
	}

	listed, err := fixture.service.ListArtifacts(t.Context(), runs.ListArtifactsRequest{ItemID: " " + item.ID + " "})
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}
	if len(listed.Artifacts) != 3 {
		t.Fatalf("ListArtifacts = %+v, want the three rows the item holds", listed.Artifacts)
	}

	byKind := make(map[string]runs.ArtifactSummary, len(listed.Artifacts))
	for _, summary := range listed.Artifacts {
		byKind[summary.Kind] = summary
	}

	written := byKind["body_html"]
	if written.Step != "generate_body" || written.Size != body.Size || written.Hash != body.Hash || written.Purged {
		t.Fatalf("body_html = %+v", written)
	}
	if written.ExpiresAt.String() != "2026-10-18T09:00:00Z" || written.CreatedAt.String() == "" {
		t.Fatalf("body_html = %+v", written)
	}
	if byKind["draft"].Size != 0 || !byKind["draft"].Purged {
		t.Fatalf("draft = %+v, want a purged row that still reports itself", byKind["draft"])
	}
	if byKind["final_report"].Purged || byKind["final_report"].ExpiresAt.String() != "" {
		t.Fatalf("final_report = %+v, want a row that never expires", byKind["final_report"])
	}

	encoded, marshalErr := json.Marshal(listed)
	if marshalErr != nil {
		t.Fatalf("Marshal: %v", marshalErr)
	}
	if strings.Contains(string(encoded), "draft</p>") || strings.Contains(string(encoded), "content") {
		t.Fatalf("the listing carries a blob: %s", encoded)
	}
	for _, key := range []string{`"kind"`, `"size"`, `"hash"`, `"purged"`, `"expiresAt"`, `"createdAt"`} {
		if !strings.Contains(string(encoded), key) {
			t.Errorf("view lacks %s: %s", key, encoded)
		}
	}
}

func TestListArtifactsAnswersAnEmptyListAndRefusesAnUnnamedItem(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	_, item := fixture.seedRun(t, run.StatusCompleted)

	listed, err := fixture.service.ListArtifacts(t.Context(), runs.ListArtifactsRequest{ItemID: item.ID})
	if err != nil {
		t.Fatalf("ListArtifacts: %v", err)
	}

	encoded, marshalErr := json.Marshal(listed)
	if marshalErr != nil {
		t.Fatalf("Marshal: %v", marshalErr)
	}
	if string(encoded) != `{"artifacts":[]}` {
		t.Fatalf("ListArtifacts = %s, want an empty array", encoded)
	}

	if _, err = fixture.service.ListArtifacts(t.Context(), runs.ListArtifactsRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("ListArtifacts = %v, want %s", err, errors.Invalid)
	}
}

func TestControlForwardsToTheEngine(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)

	if _, err := fixture.service.Pause(t.Context(), runs.PauseRequest{RunID: "r1"}); err != nil {
		t.Fatalf("Pause: %v", err)
	}
	if fixture.engine.paused != "r1" || fixture.engine.reason != run.PauseUser {
		t.Fatalf("Pause forwarded %q, %q", fixture.engine.paused, fixture.engine.reason)
	}
	if _, err := fixture.service.Pause(t.Context(), runs.PauseRequest{RunID: "r1", Reason: "needs_human"}); err != nil {
		t.Fatalf("Pause with a reason: %v", err)
	}
	if fixture.engine.reason != run.PauseNeedsHuman {
		t.Fatalf("Pause reason = %q", fixture.engine.reason)
	}

	if _, err := fixture.service.Resume(t.Context(), runs.ResumeRequest{RunID: "r1"}); err != nil {
		t.Fatalf("Resume: %v", err)
	}
	if _, err := fixture.service.Cancel(t.Context(), runs.CancelRequest{RunID: "r1"}); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	if _, err := fixture.service.RetryStep(t.Context(), runs.RetryStepRequest{ItemID: "i1"}); err != nil {
		t.Fatalf("RetryStep: %v", err)
	}
	if fixture.engine.resumed != "r1" || fixture.engine.cancelled != "r1" || fixture.engine.retried != "i1" {
		t.Fatalf("the engine saw %q, %q, %q", fixture.engine.resumed, fixture.engine.cancelled, fixture.engine.retried)
	}
}

func TestControlRejectsBadRequests(t *testing.T) {
	t.Parallel()

	fixture := newFixture(t)
	if _, err := fixture.service.Pause(t.Context(), runs.PauseRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Pause without a run = %v", err)
	}
	if _, err := fixture.service.Pause(t.Context(), runs.PauseRequest{RunID: "r", Reason: "bored"}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Pause with an unknown reason = %v", err)
	}
	if _, err := fixture.service.Resume(t.Context(), runs.ResumeRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Resume without a run = %v", err)
	}
	if _, err := fixture.service.Cancel(t.Context(), runs.CancelRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("Cancel without a run = %v", err)
	}
	if _, err := fixture.service.RetryStep(t.Context(), runs.RetryStepRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("RetryStep without an item = %v", err)
	}

	fixture.engine.failWith = errors.New(errors.Conflict, "the run finished")
	for _, call := range []func() error{
		func() error { _, err := fixture.service.Pause(t.Context(), runs.PauseRequest{RunID: "r"}); return err },
		func() error {
			_, err := fixture.service.Resume(t.Context(), runs.ResumeRequest{RunID: "r"})
			return err
		},
		func() error {
			_, err := fixture.service.Cancel(t.Context(), runs.CancelRequest{RunID: "r"})
			return err
		},
		func() error {
			_, err := fixture.service.RetryStep(t.Context(), runs.RetryStepRequest{ItemID: "i"})
			return err
		},
	} {
		if err := call(); !errors.IsCode(err, errors.Conflict) {
			t.Fatalf("the engine failure = %v", err)
		}
	}
}
