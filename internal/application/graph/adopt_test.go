package graph_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/davidmovas/postulator/internal/adapters/sqlite"
	"github.com/davidmovas/postulator/internal/adapters/sqlite/sqlitetest"
	"github.com/davidmovas/postulator/internal/application/applicationtest"
	appgraph "github.com/davidmovas/postulator/internal/application/graph"
	"github.com/davidmovas/postulator/internal/domain/finding"
	graphdomain "github.com/davidmovas/postulator/internal/domain/graph"
	"github.com/davidmovas/postulator/internal/domain/pagemap"
	"github.com/davidmovas/postulator/internal/kernel/clock"
	"github.com/davidmovas/postulator/internal/kernel/errors"
	"github.com/davidmovas/postulator/internal/kernel/id"
)

func (f *siteProposals) canonical(t *testing.T, name string, scope *string, page *pagemap.Page) graphdomain.Entity {
	t.Helper()

	entity := f.entity(t, name, scope)
	if page != nil {
		if err := sqlite.NewEntityRepo(f.store).SetCanonicalPage(t.Context(), entity.ID, &page.ID, sqlitetest.Stamp); err != nil {
			t.Fatalf("set the canonical page of %s: %v", name, err)
		}
	}
	return entity
}

func (f *siteProposals) parentEdge(t *testing.T, child, parent graphdomain.Entity) {
	t.Helper()

	edge, err := graphdomain.NewEdge(graphdomain.Edge{
		ID: id.New(), SiteID: f.siteID, FromEntityID: child.ID, ToEntityID: parent.ID, Kind: graphdomain.EdgeParent,
		Weight: 1, Source: graphdomain.SourceUser, Status: graphdomain.StatusApproved, CreatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("build the edge: %v", err)
	}
	if err = sqlite.NewEdgeRepo(f.store).Insert(t.Context(), edge); err != nil {
		t.Fatalf("insert the edge: %v", err)
	}
}

func (f *siteProposals) parentsOf(t *testing.T, entityID string) []string {
	t.Helper()

	edges, err := sqlite.NewEdgeRepo(f.store).ListBySite(t.Context(), f.siteID)
	if err != nil {
		t.Fatalf("list the edges: %v", err)
	}
	out := make([]string, 0)
	for i := range edges {
		if edges[i].Kind == graphdomain.EdgeParent && edges[i].FromEntityID == entityID && edges[i].Status == graphdomain.StatusApproved {
			out = append(out, edges[i].ToEntityID)
		}
	}
	return out
}

func (f proposeFixture) child(t *testing.T, path, title string, parent pagemap.Page) pagemap.Page {
	t.Helper()

	page := f.page(t, path, title)
	page.ParentPageID, page.ParentReason, page.WPID = new(parent.ID), pagemap.ParentJSONLD, new(int64(len(path)))
	if err := f.pages.Update(t.Context(), page); err != nil {
		t.Fatalf("place %s under %s: %v", path, parent.Path, err)
	}
	return page
}

func TestAModelProposalWithoutAParentTakesThePageTreesParent(t *testing.T) {
	t.Parallel()

	answer := `{"entities":[
		{"path":"/coffee/espresso/","name":"Espresso","intent":"","primaryKeyword":"espresso","secondaryKeywords":[],"anchors":[],"parentPath":"","relatedPaths":[]},
		{"path":"/coffee/espresso/doppio/","name":"Doppio","intent":"","primaryKeyword":"doppio","secondaryKeywords":[],"anchors":[],"parentPath":"","relatedPaths":[]},
		{"path":"/coffee/filter/","name":"Filter","intent":"","primaryKeyword":"filter","secondaryKeywords":[],"anchors":[],"parentPath":"/tea/","relatedPaths":[]}
	]}`
	f := newProposeFixture(t, &scriptedModel{replies: []string{answer}}, fixedProfiles{})
	service := appgraph.New(appgraph.Deps{
		Entities: sqlite.NewEntityRepo(f.store), Edges: f.edges, Sites: sqlite.NewSiteRepo(f.store), Pages: f.pages,
		Profiles: fixedProfiles{}, LLM: f.model, Identities: sqlite.NewPageIdentityRepo(f.store), Fields: sqlite.NewPageFieldRepo(f.store),
		Findings: sqlite.NewFindingRepo(f.store), Maps: storeMap{store: f.store}, UnitOfWork: f.store,
		Publisher: &applicationtest.Recorder{}, Clock: clock.NewFake(sqlitetest.Stamp),
	})
	coffee := f.page(t, "/coffee/", "Coffee")
	created, err := service.CreateEntity(t.Context(), appgraph.CreateEntityRequest{SiteID: f.siteID, Name: "Coffee"})
	if err != nil {
		t.Fatalf("CreateEntity: %v", err)
	}
	coffee.EntityID = new(created.Entity.ID)
	if err = f.pages.Update(t.Context(), coffee); err != nil {
		t.Fatalf("map the hub: %v", err)
	}
	espresso := f.child(t, "/coffee/espresso/", "Espresso", coffee)
	f.child(t, "/coffee/espresso/doppio/", "Doppio", espresso)
	f.child(t, "/coffee/filter/", "Filter", coffee)
	f.page(t, "/tea/", "Tea")

	out, err := service.PreviewFromPages(t.Context(), appgraph.PreviewFromPagesRequest{SiteID: f.siteID, PathPrefix: "/coffee/"})
	if err != nil {
		t.Fatalf("PreviewFromPages: %v", err)
	}
	byPath := map[string]appgraph.ProposedEntity{}
	for _, proposed := range out.Entities {
		byPath[proposed.Path] = proposed
	}
	if got := byPath["/coffee/espresso/"]; got.ParentEntityID != created.Entity.ID || got.Parent != "/coffee/" {
		t.Errorf("espresso's parent = %q / %q, want the hub and its entity", got.Parent, got.ParentEntityID)
	}
	if got := byPath["/coffee/espresso/doppio/"]; got.Parent != "/coffee/espresso/" || got.ParentEntityID != "" {
		t.Errorf("doppio's parent = %q / %q, want the espresso proposed beside it", got.Parent, got.ParentEntityID)
	}
	if got := byPath["/coffee/filter/"]; got.Parent != "/tea/" || got.ParentEntityID != "" {
		t.Errorf("a parent the model named = %q / %q, want it kept", got.Parent, got.ParentEntityID)
	}
}

func TestAGraphWriterThatMovesAParentSettlesTheTwoHierarchiesAtOnce(t *testing.T) {
	t.Parallel()

	f := newSiteProposals(t)
	serumsPage := f.onSite(t, "/serums/", "Serums", pagemap.WPPage, nil, nil)
	oilsPage := f.onSite(t, "/oils/", "Oils", pagemap.WPPage, nil, nil)
	vitcPage := f.onSite(t, "/serums/vitamin-c/", "Vitamin C", pagemap.WPPage, &serumsPage, nil)
	serums := f.canonical(t, "Serums", nil, &serumsPage)
	oils := f.canonical(t, "Oils", nil, &oilsPage)
	vitc := f.canonical(t, "Vitamin C", nil, &vitcPage)
	for _, held := range []struct {
		page   *pagemap.Page
		entity graphdomain.Entity
	}{{&serumsPage, serums}, {&oilsPage, oils}, {&vitcPage, vitc}} {
		held.page.EntityID = new(held.entity.ID)
		if err := f.pages.Update(t.Context(), *held.page); err != nil {
			t.Fatalf("map %s: %v", held.page.Path, err)
		}
	}
	subject := finding.SubjectEntityParent(vitc.ID)
	status := func() finding.Status {
		t.Helper()
		open, err := f.findings.OpenBySite(t.Context(), f.siteID, finding.EntityParentDiffers)
		if err != nil {
			t.Fatalf("OpenBySite: %v", err)
		}
		for i := range open {
			if open[i].SubjectKey == subject {
				return open[i].Status
			}
		}
		return ""
	}

	if _, err := f.service.AddEdge(t.Context(), appgraph.AddEdgeRequest{
		SiteID: f.siteID, FromEntityID: vitc.ID, ToEntityID: oils.ID, Kind: string(graphdomain.EdgeParent),
	}); err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	if got := status(); got != finding.Open {
		t.Fatalf("a parent edge the page tree contradicts left the finding %q, want it open at once", got)
	}
	if _, err := f.service.MoveEntity(t.Context(), appgraph.MoveEntityRequest{EntityID: vitc.ID, NewParentID: serums.ID}); err != nil {
		t.Fatalf("MoveEntity: %v", err)
	}
	if got := status(); got != "" {
		t.Errorf("a move to the page tree's parent left the finding %q, want it settled", got)
	}
}

func TestTheGraphAdoptsThePageTreesParentsInOnePreviewedAction(t *testing.T) {
	t.Parallel()

	f := newSiteProposals(t)
	serumsPage := f.onSite(t, "/serums/", "Serums", pagemap.WPPage, nil, nil)
	oilsPage := f.onSite(t, "/oils/", "Oils", pagemap.WPPage, nil, nil)
	serums := f.canonical(t, "Serums", nil, &serumsPage)
	oils := f.canonical(t, "Oils", nil, &oilsPage)
	vitcPage := f.onSite(t, "/serums/vitamin-c/", "Vitamin C", pagemap.WPPage, &serumsPage, nil)
	retinolPage := f.onSite(t, "/serums/retinol/", "Retinol", pagemap.WPPage, &serumsPage, nil)
	arganPage := f.onSite(t, "/oils/argan/", "Argan", pagemap.WPPage, &oilsPage, nil)
	vitc := f.canonical(t, "Vitamin C", nil, &vitcPage)
	retinol := f.canonical(t, "Retinol", nil, &retinolPage)
	argan := f.canonical(t, "Argan", &serums.ID, &arganPage)
	f.parentEdge(t, argan, serums)
	for page, entity := range map[*pagemap.Page]graphdomain.Entity{&serumsPage: serums, &oilsPage: oils, &vitcPage: vitc, &retinolPage: retinol, &arganPage: argan} {
		page.EntityID = new(entity.ID)
		if err := f.pages.Update(t.Context(), *page); err != nil {
			t.Fatalf("map %s: %v", page.Path, err)
		}
	}
	open, err := finding.New(finding.Finding{
		ID: id.New(), SiteID: f.siteID, PageID: new(vitcPage.ID), EntityID: new(vitc.ID), Code: finding.EntityParentDiffers,
		SubjectKey: finding.SubjectEntityParent(vitc.ID), Subject: json.RawMessage(`{}`), CreatedAt: sqlitetest.Stamp, UpdatedAt: sqlitetest.Stamp,
	})
	if err != nil {
		t.Fatalf("build the finding: %v", err)
	}
	if err = f.findings.Upsert(t.Context(), open); err != nil {
		t.Fatalf("raise the finding: %v", err)
	}

	preview, err := f.service.AdoptParents(t.Context(), appgraph.AdoptParentsRequest{SiteID: f.siteID})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Adopting != 3 || preview.Skipped != 0 || preview.Hash == "" || preview.Applied || len(preview.Steps.Items) != 3 {
		t.Fatalf("the preview = %+v", preview)
	}
	byName := map[string]appgraph.AdoptionView{}
	for _, step := range preview.Steps.Items {
		byName[step.Name] = step
	}
	if step := byName["Argan"]; step.To == nil || step.To.ID != oils.ID || len(step.From) != 1 || step.From[0].ID != serums.ID {
		t.Errorf("argan's step = %+v, want from Serums to Oils", step)
	}
	if got := f.parentsOf(t, vitc.ID); len(got) != 0 {
		t.Fatalf("a preview wrote %v", got)
	}

	first, err := f.service.AdoptParents(t.Context(), appgraph.AdoptParentsRequest{SiteID: f.siteID, Limit: 2})
	if err != nil || len(first.Steps.Items) != 2 || !first.Steps.HasMore || first.Hash != preview.Hash {
		t.Fatalf("a page of the preview = %+v, %v", first, err)
	}
	section, err := f.service.AdoptParents(t.Context(), appgraph.AdoptParentsRequest{SiteID: f.siteID, Under: oilsPage.ID})
	if err != nil || section.Adopting != 1 || section.Steps.Items[0].Name != "Argan" {
		t.Fatalf("the preview under /oils/ = %+v, %v", section, err)
	}

	if _, err = f.service.AdoptParents(t.Context(), appgraph.AdoptParentsRequest{SiteID: f.siteID, Apply: true, Hash: "stale"}); !errors.IsCode(err, errors.Conflict) {
		t.Fatalf("an apply under a stale hash answered %v", err)
	}
	applied, err := f.service.AdoptParents(t.Context(), appgraph.AdoptParentsRequest{SiteID: f.siteID, Apply: true, Hash: preview.Hash})
	if err != nil || !applied.Applied || applied.Adopting != 3 {
		t.Fatalf("the apply = %+v, %v", applied, err)
	}
	for _, tc := range []struct {
		entity graphdomain.Entity
		want   []string
	}{{vitc, []string{serums.ID}}, {retinol, []string{serums.ID}}, {argan, []string{oils.ID}}, {serums, []string{}}, {oils, []string{}}} {
		if got := f.parentsOf(t, tc.entity.ID); !slices.Equal(got, tc.want) {
			t.Errorf("%s's parents = %v, want %v", tc.entity.Name, got, tc.want)
		}
	}
	moved, err := sqlite.NewEntityRepo(f.store).Get(t.Context(), argan.ID)
	if err != nil || moved.ScopeID == nil || *moved.ScopeID != oils.ID {
		t.Errorf("argan's scope = %v, %v; want the oils it now sits under", moved.ScopeID, err)
	}
	if settled, getErr := f.findings.Get(t.Context(), open.ID); getErr != nil || settled.Status != finding.Resolved ||
		settled.Resolution != finding.ResolutionParentAdopted {
		t.Errorf("the finding = %+v, %v; want it resolved by the adoption", settled, getErr)
	}
	again, err := f.service.AdoptParents(t.Context(), appgraph.AdoptParentsRequest{SiteID: f.siteID})
	if err != nil || again.Adopting != 0 || len(again.Steps.Items) != 0 {
		t.Errorf("a preview after the apply = %+v, %v; want nothing left to adopt", again, err)
	}
}
