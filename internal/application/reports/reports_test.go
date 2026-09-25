package reports_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	"github.com/davidmovas/postulator/internal/application/reports"
	"github.com/davidmovas/postulator/internal/application/templates"
	"github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/domain/run"
	"github.com/davidmovas/postulator/internal/domain/template"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	kctx "github.com/davidmovas/postulator/internal/kernel/ctx"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

type fixture struct {
	service    *reports.Service
	store      *sqlite.Store
	siteID     string
	templateID string
	pages      map[string]pagemap.Page
	entities   map[string]graph.Entity
	runID      string
	itemID     string
}

func newFixture(t *testing.T) *fixture {
	t.Helper()

	store := sqlitetest.Open(t)
	owner := sqlitetest.Site(t, store, "shop")

	entityRepo := sqlite.NewEntityRepo(store)
	edgeRepo := sqlite.NewEdgeRepo(store)
	pageRepo := sqlite.NewPageRepo(store)
	linkRepo := sqlite.NewPageLinkRepo(store)
	runRepo := sqlite.NewRunRepo(store)
	itemRepo := sqlite.NewRunItemRepo(store)
	artifactRepo := sqlite.NewArtifactRepo(store)
	siteRepo := sqlite.NewSiteRepo(store)
	templateService := templates.New(sqlite.NewTemplateRepo(store), sqlite.NewLinkPolicyRepo(store),
		pageRepo, sqlite.NewEntityRepo(store), siteRepo, store, &applicationtest.Recorder{}, clock.NewFake(sqlitetest.Stamp))
	if err := templateService.EnsureSeeded(t.Context()); err != nil {
		t.Fatalf("seed the templates: %v", err)
	}
	hub := sqlitetest.Template(t, store, "Hub fixture")
	owner.Defaults.TemplateID = &hub.ID
	if err := siteRepo.Update(t.Context(), owner); err != nil {
		t.Fatalf("give the site a default template: %v", err)
	}

	f := &fixture{
		store: store, siteID: owner.ID, templateID: hub.ID,
		pages: make(map[string]pagemap.Page), entities: make(map[string]graph.Entity),
		service: reports.New(reports.Deps{
			Entities: entityRepo, Edges: edgeRepo, Pages: pageRepo, Links: linkRepo,
			Runs: runRepo, Items: itemRepo, Artifacts: artifactRepo,
			Sites: siteRepo, Specs: templateService, Policies: templateService,
		}),
	}

	parentPage := f.page(t, pageRepo, "/coffee/", pagemap.StatusPublished)
	childPage := f.page(t, pageRepo, "/coffee/espresso/", pagemap.StatusPublished)
	siblingPage := f.page(t, pageRepo, "/coffee/filter/", pagemap.StatusPlanned)
	loosePage := f.page(t, pageRepo, "/tea/", pagemap.StatusExists)

	parent := f.entity(t, entityRepo, "Coffee", 1, &parentPage.ID)
	child := f.entity(t, entityRepo, "Espresso", 0.6, &childPage.ID)
	sibling := f.entity(t, entityRepo, "Filter", 0.4, &siblingPage.ID)

	f.map_(t, pageRepo, parentPage, parent.ID)
	f.map_(t, pageRepo, childPage, child.ID)
	f.map_(t, pageRepo, siblingPage, sibling.ID)

	f.edge(t, edgeRepo, child.ID, parent.ID, graph.EdgeParent, graph.StatusApproved)
	f.edge(t, edgeRepo, sibling.ID, parent.ID, graph.EdgeParent, graph.StatusApproved)
	f.edge(t, edgeRepo, child.ID, sibling.ID, graph.EdgeRelated, graph.StatusApproved)
	f.edge(t, edgeRepo, parent.ID, sibling.ID, graph.EdgeRelated, graph.StatusProposed)

	f.link(t, linkRepo, childPage, parentPage.ID)
	f.link(t, linkRepo, siblingPage, childPage.ID)

	_ = loosePage
	return f
}

func (f *fixture) page(t *testing.T, repo *sqlite.PageRepo, path string, status pagemap.Status) pagemap.Page {
	t.Helper()

	record := pagemap.Page{
		ID: id.New(), SiteID: f.siteID, Path: path, Slug: pagemap.Slug(path), WPType: pagemap.WPPage,
		Title: path, H1: path, Status: status, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if status != pagemap.StatusPlanned {
		wpID := int64(len(f.pages) + 10)
		record.WPID = &wpID
	}
	if err := repo.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the page %s: %v", path, err)
	}
	f.pages[path] = record
	return record
}

func (f *fixture) map_(t *testing.T, repo *sqlite.PageRepo, page pagemap.Page, entityID string) {
	t.Helper()

	page.EntityID = &entityID
	if err := repo.Update(t.Context(), page); err != nil {
		t.Fatalf("map the page %s: %v", page.Path, err)
	}
	f.pages[page.Path] = page
}

func (f *fixture) entity(t *testing.T, repo *sqlite.EntityRepo, name string, score float64, pageID *string) graph.Entity {
	t.Helper()

	record := graph.Entity{
		ID: id.New(), SiteID: f.siteID, Name: name, Kind: graph.KindTopic, PrimaryKeyword: name,
		SecondaryKeywords: []string{}, Anchors: []graph.Anchor{{Text: name, Source: graph.AnchorUser, Weight: 1}},
		Source: graph.SourceUser, Score: score, CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	}
	if err := repo.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the entity %s: %v", name, err)
	}
	if pageID != nil {
		if err := repo.SetCanonicalPage(t.Context(), record.ID, pageID, sqlitetest.Stamp); err != nil {
			t.Fatalf("set the canonical page of %s: %v", name, err)
		}
		record.CanonicalPageID = pageID
	}
	if err := repo.SetScore(t.Context(), record.ID, score); err != nil {
		t.Fatalf("score the entity %s: %v", name, err)
	}
	f.entities[name] = record
	return record
}

func (f *fixture) edge(t *testing.T, repo *sqlite.EdgeRepo, from, to string, kind graph.EdgeKind, status graph.EdgeStatus) {
	t.Helper()

	if kind == graph.EdgeRelated && from > to {
		from, to = to, from
	}
	record := graph.Edge{
		ID: id.New(), SiteID: f.siteID, FromEntityID: from, ToEntityID: to, Kind: kind, Weight: 1,
		Source: graph.SourceUser, Status: status, CreatedAt: sqlitetest.Stamp,
	}
	if err := repo.Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the edge: %v", err)
	}
}

func (f *fixture) link(t *testing.T, repo *sqlite.PageLinkRepo, from pagemap.Page, toPageID string) {
	t.Helper()

	target := f.byID(toPageID)
	link := pagemap.PageLink{
		ID: id.New(), SiteID: f.siteID, FromPageID: from.ID, ToPageID: &toPageID, ToURL: target.Path,
		AnchorText: target.Title, Origin: pagemap.OriginObserved, ObservedAt: sqlitetest.Stamp,
	}
	if err := repo.ReplaceForPage(t.Context(), from.ID, []pagemap.PageLink{link}); err != nil {
		t.Fatalf("insert the link: %v", err)
	}
}

func (f *fixture) byID(pageID string) pagemap.Page {
	for path := range f.pages {
		if f.pages[path].ID == pageID {
			return f.pages[path]
		}
	}
	return pagemap.Page{}
}

func (f *fixture) withRun(t *testing.T, status run.Status, blobs map[run.ArtifactKind]string) {
	t.Helper()

	record := run.Run{
		ID: id.New(), SiteID: f.siteID, Kind: run.KindGenerate, Status: status,
		PublishMode: run.PublishDraft, CreatedBy: kctx.ActorUser,
		Targets:   []string{f.pages["/coffee/espresso/"].ID},
		Recipe:    []template.StepSpec{{Name: "resolve_context", Enabled: true}},
		Stats:     run.Stats{Items: 1, Done: 1, Tokens: 42, USD: 0.5},
		CreatedAt: sqlitetest.Stamp, DeadlineAt: sqlitetest.Stamp.Add(time.Hour),
	}
	if err := sqlite.NewRunRepo(f.store).Insert(t.Context(), record); err != nil {
		t.Fatalf("insert the run: %v", err)
	}

	finished := sqlitetest.Stamp.Add(time.Minute)
	item := run.Item{
		ID: id.New(), RunID: record.ID, SiteID: f.siteID, TargetID: f.pages["/coffee/espresso/"].ID, Status: status,
		CurrentStep: "report", Checkpoint: run.NewCheckpoint(),
		CreatedAt: sqlitetest.Stamp, UpdatedAt: finished, FinishedAt: &finished,
	}
	if err := sqlite.NewRunItemRepo(f.store).Insert(t.Context(), item); err != nil {
		t.Fatalf("insert the run item: %v", err)
	}

	artifacts := make([]run.Artifact, 0, len(blobs))
	for kind, blob := range blobs {
		artifacts = append(artifacts, run.Artifact{
			ID: id.New(), RunID: record.ID, ItemID: item.ID, Step: "report", Kind: kind,
			Blob: []byte(blob), Size: len(blob), Hash: run.HashBlob([]byte(blob)), CreatedAt: sqlitetest.Stamp,
		})
	}
	if err := sqlite.NewArtifactRepo(f.store).ReplaceStep(t.Context(), item.ID, "report", artifacts); err != nil {
		t.Fatalf("insert the artifacts: %v", err)
	}

	f.runID = record.ID
	f.itemID = item.ID
}

func TestSiteOverviewCountsTheGraphAndThePageMap(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	overview, err := f.service.SiteOverview(t.Context(), reports.SiteOverviewRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("SiteOverview: %v", err)
	}

	if overview.Entities != (reports.EntityTotals{Total: 3, WithCanonicalPage: 3, WithPublishedPage: 2}) {
		t.Errorf("entities = %+v", overview.Entities)
	}
	if overview.Pages.Total != 4 || overview.Pages.Unmapped != 1 {
		t.Errorf("pages = %+v", overview.Pages)
	}
	if overview.Pages.ByStatus["published"] != 2 || overview.Pages.ByStatus["planned"] != 1 {
		t.Errorf("pages by status = %+v", overview.Pages.ByStatus)
	}
	if overview.Pages.Orphans != 1 {
		t.Errorf("orphans = %d, want the one page on the site nothing links to, not the planned one", overview.Pages.Orphans)
	}
	if overview.Edges.Approved != 3 || overview.Edges.Realized != 2 {
		t.Errorf("edges = %+v", overview.Edges)
	}
	if len(overview.Depth) != 2 || overview.Depth[0] != (reports.DepthBucket{Depth: 0, Pages: 2}) {
		t.Errorf("depth = %+v", overview.Depth)
	}
	if overview.Depth[1] != (reports.DepthBucket{Depth: 1, Pages: 2}) {
		t.Errorf("depth = %+v", overview.Depth)
	}
	if len(overview.Top) != 3 || overview.Top[0].Name != "Coffee" || overview.Top[0].Path != "/coffee/" {
		t.Errorf("top = %+v", overview.Top)
	}
	if overview.Top[0].Score < overview.Top[1].Score {
		t.Errorf("top is not sorted by score: %+v", overview.Top)
	}
}

func TestSiteOverviewRefusesAnEmptySite(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	if _, err := f.service.SiteOverview(t.Context(), reports.SiteOverviewRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
}

func TestPageReportReadsTheNewestItem(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withRun(t, run.StatusCompleted, map[run.ArtifactKind]string{
		run.ArtifactValidationReport: `{"score":0.9}`,
		run.ArtifactJudgeReport:      `{"score":0.8}`,
		run.ArtifactPublishResult:    `{"wpId":7}`,
		run.ArtifactRelinkResult:     `{"linked":1}`,
	})

	report, err := f.service.PageReport(t.Context(), reports.PageReportRequest{
		PageID: f.pages["/coffee/espresso/"].ID,
	})
	if err != nil {
		t.Fatalf("PageReport: %v", err)
	}
	if report.RunID != f.runID || report.ItemID != f.itemID || report.Status != "completed" {
		t.Fatalf("report = %+v", report)
	}
	if report.Path != "/coffee/espresso/" || report.FinishedAt == nil {
		t.Fatalf("report = %+v", report)
	}
	for name, blob := range map[string]json.RawMessage{
		"validation": report.Validation, "judge": report.Judge,
		"publish": report.Publish, "relink": report.Relink,
	} {
		if len(blob) == 0 {
			t.Errorf("the %s artifact is missing", name)
		}
	}
}

func TestPageReportReportsWhatItCannotFind(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	cases := []struct {
		name string
		id   string
		want errors.Code
	}{
		{name: "no page", id: "  ", want: errors.Invalid},
		{name: "the page is gone", id: id.New(), want: errors.NotFound},
		{name: "no run has touched it", id: f.pages["/tea/"].ID, want: errors.NotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if _, err := f.service.PageReport(t.Context(), reports.PageReportRequest{PageID: tc.id}); !errors.IsCode(err, tc.want) {
				t.Fatalf("code = %q, want %q (err %v)", errors.CodeOf(err), tc.want, err)
			}
		})
	}
}

func TestRunReportCarriesTheFinalReportOfEveryItem(t *testing.T) {
	t.Parallel()

	f := newFixture(t)
	f.withRun(t, run.StatusCompleted, map[run.ArtifactKind]string{
		run.ArtifactFinalReport: `{"score":0.95,"errors":0}`,
	})

	report, err := f.service.RunReport(t.Context(), reports.RunReportRequest{RunID: f.runID})
	if err != nil {
		t.Fatalf("RunReport: %v", err)
	}
	if report.Status != "completed" || report.Kind != "generate" || report.SiteID != f.siteID {
		t.Fatalf("report = %+v", report)
	}
	if report.Stats != (reports.RunStats{Items: 1, Done: 1, Tokens: 42, USD: 0.5}) {
		t.Fatalf("stats = %+v", report.Stats)
	}
	if len(report.Items) != 1 || string(report.Items[0].Report) != `{"score":0.95,"errors":0}` {
		t.Fatalf("items = %+v", report.Items)
	}
}

func TestRunReportReportsWhatItCannotFind(t *testing.T) {
	t.Parallel()

	f := newFixture(t)

	if _, err := f.service.RunReport(t.Context(), reports.RunReportRequest{}); !errors.IsCode(err, errors.Invalid) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.Invalid)
	}
	if _, err := f.service.RunReport(t.Context(), reports.RunReportRequest{RunID: id.New()}); !errors.IsCode(err, errors.NotFound) {
		t.Fatalf("code = %q, want %q", errors.CodeOf(err), errors.NotFound)
	}
}
