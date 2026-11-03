package sqlite_test

import (
	"encoding/json"
	"slices"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/domain/agent"
	"github.com/davidmovas/postulator/internal/domain/finding"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/site"
	"github.com/davidmovas/postulator/internal/domain/template"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type plantedSite struct {
	id      string
	pages   []string
	running string
}

func plantSiteData(t *testing.T, store *sqlite.Store, name string) plantedSite {
	t.Helper()
	ctx := t.Context()
	owner := sqlitetest.Site(t, store, name)
	hub := sqlitetest.Page(t, store, owner.ID, "/shoes/")
	child := sqlitetest.Page(t, store, owner.ID, "/shoes/boots/")
	plantGraph(t, store, owner.ID, []scopedName{
		{name: "Shoes"}, {name: "Bags"}, {name: "Accessories", parent: "Shoes"}, {name: "Accessories", parent: "Bags"},
	})
	entities, err := sqlite.NewEntityRepo(store).ListBySite(ctx, owner.ID)
	if err != nil {
		t.Fatalf("list the entities: %v", err)
	}
	pages := sqlite.NewPageRepo(store)
	for i, pageID := range []string{hub.ID, child.ID} {
		page, getErr := pages.Get(ctx, pageID)
		if getErr != nil {
			t.Fatalf("get the page: %v", getErr)
		}
		page.EntityID = &entities[i].ID
		if updateErr := pages.Update(ctx, page); updateErr != nil {
			t.Fatalf("map the page: %v", updateErr)
		}
	}
	if err = sqlite.NewEdgeRepo(store).Insert(ctx, edgeBetween(t, owner.ID, entities[0].ID, entities[1].ID,
		graph.EdgeRelated, graph.StatusApproved, sqlitetest.Stamp)); err != nil {
		t.Fatalf("insert the edge: %v", err)
	}
	if err = sqlite.NewTopicBuildRepo(store).Record(ctx, owner.ID, "page:"+hub.ID, sqlitetest.Stamp); err != nil {
		t.Fatalf("record the topic: %v", err)
	}
	for _, pageID := range []*string{&child.ID, nil} {
		record, newErr := finding.New(finding.Finding{
			ID: id.New(), SiteID: owner.ID, PageID: pageID, Code: finding.PageWithoutEntity, SubjectKey: id.New(),
			Subject: json.RawMessage(`{}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
		})
		if newErr != nil {
			t.Fatalf("finding.New: %v", newErr)
		}
		if upsertErr := sqlite.NewFindingRepo(store).Upsert(ctx, record); upsertErr != nil {
			t.Fatalf("Upsert: %v", upsertErr)
		}
	}
	planted := plantedSite{id: owner.ID, pages: []string{hub.ID, child.ID}}
	runs := sqlite.NewRunRepo(store)
	for i, status := range []run.Status{run.StatusRunning, run.StatusWaiting, run.StatusCompleted} {
		record := run.Run{
			ID: id.New(), SiteID: owner.ID, Kind: run.KindGenerate, Status: status, Targets: []string{hub.ID},
			Recipe:     []template.StepSpec{{Name: "generate_body", Enabled: true}},
			TemplateID: id.New(), TemplateVersion: 1, PublishMode: run.PublishDraft,
			CreatedBy: kctx.ActorUser, DeadlineAt: sqlitetest.Stamp.Add(time.Hour), CreatedAt: sqlitetest.Stamp.Add(time.Duration(i) * time.Minute),
		}
		if insertErr := runs.Insert(ctx, record); insertErr != nil {
			t.Fatalf("insert the run: %v", insertErr)
		}
		if status == run.StatusRunning {
			planted.running = record.ID
		}
	}
	conversation, err := agent.NewConversation(agent.Conversation{
		ID: id.New(), SiteID: &owner.ID, Title: "plan the hub", Mode: agent.ModeConfirm,
		CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("NewConversation: %v", err)
	}
	if err = sqlite.NewConversationRepo(store).Insert(ctx, conversation); err != nil {
		t.Fatalf("insert the conversation: %v", err)
	}
	return planted
}

var plantedHeld = site.Held{Pages: 2, Findings: 2, Entities: 4, Edges: 1, Runs: 3, Unfinished: 1, Chats: 1}

func TestSiteDataRepoCountsWhatASiteHoldsAndWhatKeepsItBusy(t *testing.T) {
	t.Parallel()

	store := sqlitetest.Open(t)
	planted := plantSiteData(t, store, "shop")
	empty := sqlitetest.Site(t, store, "empty")
	repo := sqlite.NewSiteDataRepo(store)

	cases := []struct {
		name   string
		siteID string
		held   site.Held
		busy   []string
	}{
		{name: "a site with a graph, runs and a chat", siteID: planted.id, held: plantedHeld, busy: []string{planted.running}},
		{name: "a site that holds nothing", siteID: empty.ID, held: site.Held{}, busy: []string{}},
	}
	for _, tc := range cases {
		held, err := repo.Counts(t.Context(), tc.siteID)
		if err != nil || held != tc.held {
			t.Errorf("%s: Counts = %+v, %v; want %+v", tc.name, held, err, tc.held)
		}
		busy, err := repo.Busy(t.Context(), tc.siteID)
		if err != nil || !slices.Equal(busy, tc.busy) {
			t.Errorf("%s: Busy = %v, %v; want %v", tc.name, busy, err, tc.busy)
		}
	}
}

func TestSiteDataRepoClearsTheChosenGroupsOfOneSite(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		what   site.Clearing
		held   site.Held
		pages  int
		mapped int
		topics int
	}{
		{name: "the graph", what: site.Clearing{Graph: true},
			held:  site.Held{Pages: 2, Findings: 2, Runs: 3, Unfinished: 1, Chats: 1},
			pages: 2, mapped: 0, topics: 0},
		{name: "the history", what: site.Clearing{History: true},
			held:  site.Held{Pages: 2, Findings: 2, Entities: 4, Edges: 1},
			pages: 2, mapped: 2, topics: 1},
		{name: "the model with its history", what: site.Clearing{Model: true, History: true},
			held:  site.Held{Entities: 4, Edges: 1},
			pages: 0, topics: 1},
		{name: "everything", what: site.Clearing{Model: true, Graph: true, History: true},
			held: site.Held{}, pages: 0, topics: 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			store := sqlitetest.Open(t)
			planted := plantSiteData(t, store, "shop")
			kept := plantSiteData(t, store, "kept")
			repo := sqlite.NewSiteDataRepo(store)

			if err := repo.Clear(t.Context(), planted.id, tc.what); err != nil {
				t.Fatalf("Clear = %v", err)
			}
			if held, err := repo.Counts(t.Context(), planted.id); err != nil || held != tc.held {
				t.Errorf("Counts after = %+v, %v; want %+v", held, err, tc.held)
			}
			if held, err := repo.Counts(t.Context(), kept.id); err != nil || held != plantedHeld {
				t.Errorf("the other site holds %+v, %v; want it untouched %+v", held, err, plantedHeld)
			}
			if got := countPages(t, store, planted.id); got != tc.pages {
				t.Errorf("pages = %d, want %d", got, tc.pages)
			}
			if got := countMapped(t, store, planted.id); got != tc.mapped {
				t.Errorf("pages carrying an entity = %d, want %d", got, tc.mapped)
			}
			topics, err := sqlite.NewTopicBuildRepo(store).ListBySite(t.Context(), planted.id)
			if err != nil || len(topics) != tc.topics {
				t.Errorf("topic builds = %v, %v; want %d", topics, err, tc.topics)
			}
		})
	}
}

func countPages(t *testing.T, store *sqlite.Store, siteID string) int {
	t.Helper()
	listed, err := sqlite.NewPageRepo(store).ListBySite(t.Context(), siteID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	return len(listed)
}

func countMapped(t *testing.T, store *sqlite.Store, siteID string) int {
	t.Helper()
	listed, err := sqlite.NewPageRepo(store).ListBySite(t.Context(), siteID)
	if err != nil {
		t.Fatalf("list the pages: %v", err)
	}
	mapped := 0
	for i := range listed {
		if listed[i].EntityID != nil {
			mapped++
		}
	}
	return mapped
}
